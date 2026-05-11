package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path"
	"strconv"

	"github.com/fvdveen/generative_agents/simulation_server/llm/openai"
	"github.com/fvdveen/generative_agents/simulation_server/logging"
	simulationloader "github.com/fvdveen/generative_agents/simulation_server/simulation_loader"
	"github.com/joho/godotenv"
)

type Config struct {
	SimulationDir string
	MazeDir       string
	LogDir        string
	BackupDir     string

	SimulationName string
	SimulationMaze string

	TextModelURL string
	TextModelKey string
	TextModel    string

	EmbeddingURL   string
	EmbeddingKey   string
	EmbeddingModel string

	BackupInterval int

	LogToFile    bool
	LogLevel     slog.Level
	LogLevelFile slog.Level
}

func parseLogLevel(s, def string) slog.Level {
	if s == "" {
		s = def
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		panic(fmt.Sprintf("invalid log level %q: %v", s, err))
	}
	return l
}

func RetryPanic(fn func(), retries int) error {
	var last any

	for i := 0; i < retries; i++ {
		var pan any

		func() {
			defer func() {
				pan = recover()
			}()
			fn()
		}()

		if pan == nil {
			return nil
		}

		last = pan
	}

	log.Fatalf("function panicked after %d retries: %v", retries, last)
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		panic(fmt.Sprintf("Could not load .env file: %v", err))
	}

	var backupInterval int
	if str := os.Getenv("BACKUP_INTERVAL"); str != "" {
		var err error
		if backupInterval, err = strconv.Atoi(str); err != nil {
			panic(fmt.Sprintf("Coult not convert %q to int: %v", str, err))
		}
	}

	logLevel := parseLogLevel(os.Getenv("LOG_LEVEL"), "INFO")
	logLevelFileStr := os.Getenv("LOG_LEVEL_FILE")
	if logLevelFileStr == "" {
		logLevelFileStr = os.Getenv("LOG_LEVEL")
	}
	logLevelFile := parseLogLevel(logLevelFileStr, "INFO")

	logToFile := true
	if v := os.Getenv("LOG_TO_FILE"); v == "false" || v == "0" {
		logToFile = false
	}

	conf := Config{
		SimulationDir: os.Getenv("SIMULATION_DIR"),
		MazeDir:       os.Getenv("MAZE_DIR"),
		LogDir:        os.Getenv("LOG_DIR"),
		BackupDir:     os.Getenv("BACKUP_DIR"),

		SimulationName: os.Getenv("SIMULATION_NAME"),
		SimulationMaze: os.Getenv("SIMULATION_MAZE"),

		TextModelURL: os.Getenv("TEXT_MODEL_URL"),
		TextModelKey: os.Getenv("TEXT_MODEL_KEY"),
		TextModel:    os.Getenv("TEXT_MODEL_LLM"),

		EmbeddingKey:   os.Getenv("EMBEDDING_KEY"),
		EmbeddingURL:   os.Getenv("EMBEDDING_URL"),
		EmbeddingModel: os.Getenv("EMBEDDING_MODEL"),

		BackupInterval: backupInterval,

		LogToFile:    logToFile,
		LogLevel:     logLevel,
		LogLevelFile: logLevelFile,
	}

	rl, err := logging.NewRunLogs(logging.Config{
		BaseDir:      path.Join(conf.LogDir, conf.SimulationName),
		LogToFile:    conf.LogToFile,
		FileLevel:    conf.LogLevelFile,
		ConsoleLevel: conf.LogLevel,
	})
	if err != nil {
		panic(fmt.Sprintf("Could not react logger: %v", err))
	}
	defer func() { _ = rl.Close() }()
	defer logging.RecoverAndLog(rl.Log, rl.Sync)

	clientOpts := []openai.ClientOpt{openai.WithAPIKey(conf.TextModelKey), openai.WithLogger(rl.Log)}
	if conf.TextModelURL != "" {
		clientOpts = append(clientOpts, openai.WithURL(conf.TextModelURL))
	}
	if conf.TextModel != "" {
		clientOpts = append(clientOpts, openai.WithTextModel(conf.TextModel))
	}
	client := openai.New(clientOpts...)

	embedderOpts := []openai.ClientOpt{openai.WithAPIKey(conf.EmbeddingKey), openai.WithLogger(rl.Log)}
	if conf.EmbeddingURL != "" {
		embedderOpts = append(embedderOpts, openai.WithURL(conf.EmbeddingURL))
	}
	if conf.EmbeddingModel != "" {
		embedderOpts = append(embedderOpts, openai.WithEmbeddingsModel(conf.EmbeddingModel))
	}
	embedder := openai.New(embedderOpts...)

	RetryPanic(func() {
		sim, err := simulationloader.LoadSimulation(path.Join(conf.SimulationDir, conf.SimulationName), conf.MazeDir, embedder, client, rl.Log)
		if err != nil {
			panic(fmt.Sprintf("Could not load maze: %v\n", err))
		}

		storage := simulationloader.FileStorage{
			SimulationsFolder: conf.SimulationDir,
			Simulation:        conf.SimulationName,
			Maze:              conf.SimulationMaze,
			BackupFolder:      conf.BackupDir,
		}

		sim.Storage = &storage

		sim.BackupInterval = conf.BackupInterval
		if err := sim.Run(100000); err != nil {
			panic(fmt.Sprintf("Could not run simulation: %v", err))
		}
	}, 999999)
}
