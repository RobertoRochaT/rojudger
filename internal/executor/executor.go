package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/RobertoRochaT/rojudger/internal/config"
	"github.com/RobertoRochaT/rojudger/internal/models"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Executor maneja la ejecución de código en contenedores Docker
type Executor struct {
	client      *client.Client
	config      *config.Config
	rateLimiter chan struct{} // Canal para limitar ejecuciones concurrentes
}

// NewExecutor crea una nueva instancia del executor
func NewExecutor(cfg *config.Config) (*Executor, error) {
	// Crear cliente Docker
	cli, err := client.NewClientWithOpts(
		client.WithHost(cfg.DockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Verificar que Docker está disponible
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("docker daemon not available: %w", err)
	}

	// Crear canal para limitar concurrencia
	rateLimiter := make(chan struct{}, cfg.ExecutorMaxConcurrent)

	log.Printf("Executor initialized (max concurrent: %d)", cfg.ExecutorMaxConcurrent)

	return &Executor{
		client:      cli,
		config:      cfg,
		rateLimiter: rateLimiter,
	}, nil
}

// Execute ejecuta el código en un contenedor Docker aislado
func (e *Executor) Execute(ctx context.Context, submission *models.Submission, language *models.Language) models.ExecutionResult {
	// Limitar concurrencia
	e.rateLimiter <- struct{}{}
	defer func() { <-e.rateLimiter }()

	result := models.ExecutionResult{
		ExitCode: -1,
	}

	// Crear contexto con timeout
	execCtx, cancel := context.WithTimeout(ctx, e.config.ExecutorTimeout)
	defer cancel()

	startTime := time.Now()

	// Si el lenguaje requiere compilación, compilar primero
	if language.IsCompiled {
		compileResult := e.compile(execCtx, submission, language)
		result.CompileOut = compileResult.Stdout + compileResult.Stderr

		if compileResult.ExitCode != 0 {
			result.Stderr = compileResult.Stderr
			result.ExitCode = compileResult.ExitCode
			result.Error = "Compilation failed"
			return result
		}
	}

	// Ejecutar el código
	containerID, err := e.createContainer(execCtx, submission, language)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create container: %v", err)
		return result
	}

	// Asegurar limpieza del contenedor
	defer e.cleanup(containerID)

	// Iniciar contenedor
	if err := e.client.ContainerStart(execCtx, containerID, types.ContainerStartOptions{}); err != nil {
		result.Error = fmt.Sprintf("Failed to start container: %v", err)
		return result
	}

	// Enviar stdin si existe
	if submission.Stdin != "" {
		if err := e.writeStdin(execCtx, containerID, submission.Stdin); err != nil {
			log.Printf("Warning: failed to write stdin: %v", err)
		}
	}

	// Esperar a que termine o timeout
	statusCh, errCh := e.client.ContainerWait(execCtx, containerID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			result.Error = fmt.Sprintf("Container wait error: %v", err)
			return result
		}
	case status := <-statusCh:
		result.ExitCode = int(status.StatusCode)
	case <-execCtx.Done():
		// Timeout - forzar detención del contenedor
		result.TimedOut = true
		e.client.ContainerStop(context.Background(), containerID, container.StopOptions{})
	}

	// Calcular tiempo de ejecución
	result.Time = time.Since(startTime).Seconds()

	// Obtener logs (stdout y stderr)
	stdout, stderr, err := e.getLogs(context.Background(), containerID)
	if err != nil {
		log.Printf("Warning: failed to get logs: %v", err)
	}

	result.Stdout = stdout
	result.Stderr = stderr

	// Obtener estadísticas de memoria
	stats, err := e.getStats(context.Background(), containerID)
	if err == nil {
		result.Memory = stats.MemoryUsageKB
	}

	return result
}

// createContainer crea un contenedor Docker con límites de recursos
func (e *Executor) createContainer(ctx context.Context, submission *models.Submission, language *models.Language) (string, error) {
	// Preparar el comando de ejecución
	cmd := e.buildExecuteCommand(submission, language)

	// Configurar límites de recursos
	resources := container.Resources{
		Memory:   parseMemoryLimit(e.config.ExecutorMemoryLimit), // 256MB por defecto
		NanoCPUs: parseCPULimit(e.config.ExecutorCPULimit),       // 0.5 CPUs por defecto
	}

	// Configuración del contenedor
	containerConfig := &container.Config{
		Image:           language.DockerImage,
		Cmd:             cmd,
		Tty:             false,
		AttachStdin:     true,
		AttachStdout:    true,
		AttachStderr:    true,
		OpenStdin:       true,
		StdinOnce:       true,
		WorkingDir:      "/workspace",
		NetworkDisabled: true, // Deshabilitar red por seguridad
	}

	hostConfig := &container.HostConfig{
		Resources:      resources,
		AutoRemove:     false, // Removemos manualmente después de obtener logs
		NetworkMode:    "none",
		ReadonlyRootfs: false,           // Algunos lenguajes necesitan escribir archivos temporales
		CapDrop:        []string{"ALL"}, // Eliminar todas las capabilities por seguridad
		SecurityOpt:    []string{"no-new-privileges"},
	}

	// Crear contenedor
	resp, err := e.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		"",
	)

	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// buildExecuteCommand construye el comando para ejecutar el código
func (e *Executor) buildExecuteCommand(submission *models.Submission, language *models.Language) []string {
	// Para lenguajes interpretados, crear archivo y ejecutar
	filename := "main" + language.Extension

	// Comando para crear el archivo con el código fuente
	createFile := fmt.Sprintf("echo %q > /workspace/%s", submission.SourceCode, filename)

	// Comando de ejecución
	executeCmd := strings.ReplaceAll(language.ExecuteCmd, "{file}", filename)

	// Combinar comandos
	fullCmd := []string{
		"sh", "-c",
		fmt.Sprintf("%s && cd /workspace && %s", createFile, executeCmd),
	}

	return fullCmd
}

// writeStdin escribe datos al stdin del contenedor
func (e *Executor) writeStdin(ctx context.Context, containerID string, stdin string) error {
	attachResp, err := e.client.ContainerAttach(ctx, containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
	})
	if err != nil {
		return err
	}
	defer attachResp.Close()

	_, err = attachResp.Conn.Write([]byte(stdin))
	if err != nil {
		return err
	}

	// Cerrar stdin para señalar que terminamos
	return attachResp.CloseWrite()
}

// getLogs obtiene stdout y stderr del contenedor
func (e *Executor) getLogs(ctx context.Context, containerID string) (string, string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}

	logs, err := e.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", "", err
	}
	defer logs.Close()

	// Docker multiplexa stdout/stderr en un stream
	// Necesitamos demultiplexarlo
	var stdout, stderr strings.Builder

	// Buffer temporal
	buf := make([]byte, 8)

	for {
		// Leer header (8 bytes)
		n, err := logs.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil || n < 8 {
			break
		}

		// byte 0: stream type (1=stdout, 2=stderr)
		streamType := buf[0]
		// bytes 4-7: tamaño del payload
		size := int(buf[4])<<24 | int(buf[5])<<16 | int(buf[6])<<8 | int(buf[7])

		// Leer payload
		payload := make([]byte, size)
		_, err = io.ReadFull(logs, payload)
		if err != nil {
			break
		}

		// Escribir al stream correspondiente
		if streamType == 1 {
			stdout.Write(payload)
		} else if streamType == 2 {
			stderr.Write(payload)
		}
	}

	return stdout.String(), stderr.String(), nil
}

// ContainerStats representa estadísticas del contenedor
type ContainerStats struct {
	MemoryUsageKB int
}

// getStats obtiene estadísticas del contenedor
func (e *Executor) getStats(ctx context.Context, containerID string) (ContainerStats, error) {
	stats := ContainerStats{}

	statsResp, err := e.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return stats, err
	}
	defer statsResp.Body.Close()

	var containerStats types.StatsJSON
	if err := json.NewDecoder(statsResp.Body).Decode(&containerStats); err != nil {
		return stats, err
	}

	// Convertir bytes a KB
	stats.MemoryUsageKB = int(containerStats.MemoryStats.Usage / 1024)

	return stats, nil
}

// compile compila el código (para lenguajes compilados como C, C++, Go)
func (e *Executor) compile(ctx context.Context, submission *models.Submission, language *models.Language) models.ExecutionResult {
	result := models.ExecutionResult{
		ExitCode: -1,
	}

	// Similar a Execute pero usando CompileCmd
	filename := "main" + language.Extension
	createFile := fmt.Sprintf("echo %q > /workspace/%s", submission.SourceCode, filename)
	compileCmd := strings.ReplaceAll(language.CompileCmd, "{file}", filename)

	fullCmd := []string{
		"sh", "-c",
		fmt.Sprintf("%s && cd /workspace && %s", createFile, compileCmd),
	}

	containerConfig := &container.Config{
		Image:      language.DockerImage,
		Cmd:        fullCmd,
		WorkingDir: "/workspace",
	}

	hostConfig := &container.HostConfig{
		AutoRemove:  false,
		NetworkMode: "none",
	}

	resp, err := e.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create compile container: %v", err)
		return result
	}

	defer e.cleanup(resp.ID)

	if err := e.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		result.Error = fmt.Sprintf("Failed to start compile container: %v", err)
		return result
	}

	statusCh, errCh := e.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			result.Error = err.Error()
			return result
		}
	case status := <-statusCh:
		result.ExitCode = int(status.StatusCode)
	}

	stdout, stderr, _ := e.getLogs(context.Background(), resp.ID)
	result.Stdout = stdout
	result.Stderr = stderr

	return result
}

// cleanup limpia el contenedor
func (e *Executor) cleanup(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Detener contenedor si está corriendo
	e.client.ContainerStop(ctx, containerID, container.StopOptions{})

	// Remover contenedor
	e.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	})
}

// Close cierra el cliente Docker
func (e *Executor) Close() error {
	return e.client.Close()
}

// Helper functions

func parseMemoryLimit(limit string) int64 {
	// Parsear strings como "256m", "1g"
	// Por simplicidad, asumimos que viene en formato correcto
	if strings.HasSuffix(limit, "m") || strings.HasSuffix(limit, "M") {
		var mb int64
		fmt.Sscanf(limit, "%d", &mb)
		return mb * 1024 * 1024 // Convertir a bytes
	}
	return 256 * 1024 * 1024 // Default 256MB
}

func parseCPULimit(limit string) int64 {
	// CPU limit como fracción (0.5 = 50% de un CPU)
	var cpu float64
	fmt.Sscanf(limit, "%f", &cpu)
	return int64(cpu * 1e9) // Convertir a nanocpus
}
