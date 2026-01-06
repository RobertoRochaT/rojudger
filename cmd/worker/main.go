package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/RobertoRochaT/rojudger/internal/config"
	"github.com/RobertoRochaT/rojudger/internal/database"
	"github.com/RobertoRochaT/rojudger/internal/executor"
	"github.com/RobertoRochaT/rojudger/internal/queue"
)

func main() {
	log.Println("🚀 Starting ROJUDGER Worker...")

	// Cargar configuración
	cfg := config.Load()

	// Conectar a la base de datos
	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Conectar a Redis queue
	q, err := queue.NewQueue(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to queue: %v", err)
	}
	defer q.Close()

	// Crear executor
	exec, err := executor.NewExecutor(cfg)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}
	defer exec.Close()

	// Número de workers concurrentes
	numWorkers := cfg.ExecutorMaxConcurrent
	if numWorkers == 0 {
		numWorkers = 5
	}

	log.Printf("Starting %d concurrent workers", numWorkers)

	// Canal para señales de sistema
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Context para cancelación
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WaitGroup para esperar a que todos los workers terminen
	var wg sync.WaitGroup

	// Iniciar workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID, db, q, exec)
		}(i + 1)
	}

	log.Println("✅ Workers started. Press Ctrl+C to stop.")

	// Esperar señal de terminación
	<-sigChan
	log.Println("⚠️  Shutdown signal received. Stopping workers...")

	// Cancelar context para detener workers
	cancel()

	// Esperar a que todos los workers terminen
	wg.Wait()

	log.Println("✅ All workers stopped. Goodbye!")
}

func runWorker(ctx context.Context, workerID int, db *database.DB, q *queue.Queue, exec *executor.Executor) {
	log.Printf("Worker #%d started", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker #%d stopping...", workerID)
			return
		default:
			// Intentar obtener un trabajo (esperar hasta 5 segundos)
			job, err := q.Dequeue(ctx, 5*time.Second)
			if err != nil {
				log.Printf("Worker #%d: Error dequeuing job: %v", workerID, err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Si no hay trabajos, continuar esperando
			if job == nil {
				continue
			}

			// Procesar el trabajo
			log.Printf("Worker #%d: Processing job %s", workerID, job.SubmissionID)
			
			if err := processSubmission(ctx, workerID, job.SubmissionID, db, exec, q); err != nil {
				log.Printf("Worker #%d: Error processing job %s: %v", workerID, job.SubmissionID, err)
				q.MarkFailed(ctx, job.SubmissionID, false)
			} else {
				log.Printf("Worker #%d: Job %s completed successfully", workerID, job.SubmissionID)
				q.MarkComplete(ctx, job.SubmissionID)
			}
		}
	}
}

func processSubmission(ctx context.Context, workerID int, submissionID string, db *database.DB, exec *executor.Executor, q *queue.Queue) error {
	// 1. Obtener submission de la base de datos
	submission, err := db.GetSubmission(submissionID)
	if err != nil {
		return err
	}

	// 2. Actualizar estado a "processing"
	submission.Status = "processing"
	if err := db.UpdateSubmission(submission); err != nil {
		return err
	}

	// 3. Obtener información del lenguaje
	language, err := db.GetLanguage(submission.LanguageID)
	if err != nil {
		return err
	}

	// 4. Ejecutar el código
	log.Printf("Worker #%d: Executing code for submission %s (language: %s)", 
		workerID, submissionID, language.DisplayName)
	
	result := exec.Execute(ctx, submission, language)

	// 5. Actualizar submission con los resultados
	now := time.Now()
	submission.FinishedAt = &now
	submission.Stdout = result.Stdout
	submission.Stderr = result.Stderr
	submission.ExitCode = result.ExitCode
	submission.Time = result.Time
	submission.Memory = result.Memory
	submission.CompileOut = result.CompileOut

	if result.TimedOut {
		submission.Status = "timeout"
		submission.Message = "Execution timed out"
	} else if result.Error != "" {
		submission.Status = "error"
		submission.Message = result.Error
	} else {
		submission.Status = "completed"
	}

	// 6. Guardar en base de datos
	if err := db.UpdateSubmission(submission); err != nil {
		return err
	}

	return nil
}
