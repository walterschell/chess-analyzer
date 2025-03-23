package chessanalysis

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
)

var log = slog.Default().With("package", "chessanalysis")

const StartingPositionScore = 0.11

type StockfishEngine struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Scanner
	ready     bool
	mutex     sync.Mutex
	responses chan string
}

type AnalysisResult struct {
	Score         float64
	WinProb       float64
	DrawProb      float64
	LossProb      float64
	BestMove      string
	BestMoveScore float64
}

// NewStockfishEngine creates and initializes a new Stockfish engine instance
func NewStockfishEngine() (*StockfishEngine, error) {
	cmd := exec.Command("stockfish")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	engine := &StockfishEngine{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    bufio.NewScanner(stdout),
		responses: make(chan string, 100),
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Stockfish: %v", err)
	}

	// Initialize engine
	go engine.readOutput()
	if err := engine.initialize(); err != nil {
		engine.Close()
		return nil, err
	}

	return engine, nil
}

// initialize sets up the Stockfish engine with UCI protocol
func (e *StockfishEngine) initialize() error {
	e.sendCommand("uci")
	e.sendCommand("setoption name Hash value 128")
	e.sendCommand("setoption name Threads value 4")
	e.sendCommand("setoption name Ponder value false")
	e.sendCommand("setoption name UCI_ShowWDL value true")
	e.sendCommand("isready")

	// Wait for readyok
	for response := range e.responses {
		if strings.Contains(response, "readyok") {
			e.ready = true
			return nil
		}
	}
	return fmt.Errorf("engine initialization failed")
}

// sendCommand sends a command to the Stockfish engine
func (e *StockfishEngine) sendCommand(cmd string) error {
	log.Info("sending command", "command", cmd)
	e.mutex.Lock()
	defer e.mutex.Unlock()
	_, err := fmt.Fprintln(e.stdin, cmd)
	return err
}

// readOutput continuously reads engine output
func (e *StockfishEngine) readOutput() {
	for e.stdout.Scan() {
		response := e.stdout.Text()
		log.Info("received response", "response", response)
		e.responses <- response
	}
	close(e.responses)
}

// analyzePosition analyzes a position at the given depth
func (e *StockfishEngine) analyzeLastMove(moves []string, depth int) (*AnalysisResult, error) {
	if !e.ready {
		return nil, fmt.Errorf("engine not ready")
	}
	if len(moves) == 0 {
		return nil, fmt.Errorf("no moves provided")
	}

	// Get the last move
	lastMove := moves[len(moves)-1]

	// Set up position before the last move
	if len(moves) > 1 {
		e.sendCommand(fmt.Sprintf("position startpos moves %s", strings.Join(moves[:len(moves)-1], " ")))
	} else {
		e.sendCommand("position startpos")
	}

	// First analysis: Find what the best move would have been from the position before the last move
	e.sendCommand(fmt.Sprintf("go depth %d", depth))
	lastScore := 0.0
	bestMove := ""
	var bestWinProb, bestDrawProb, bestLossProb float64

	for response := range e.responses {
		if strings.Contains(response, "score cp ") {
			parts := strings.Split(response, "score cp ")
			if len(parts) > 1 {
				fmt.Sscanf(parts[1], "%f", &lastScore)
			}
		}
		if strings.Contains(response, " wdl ") {
			// Parse WDL statistics (win/draw/loss in permille)
			parts := strings.Split(response, " wdl ")
			if len(parts) > 1 {
				var win, draw, loss int
				fmt.Sscanf(parts[1], "%d %d %d", &win, &draw, &loss)
				bestWinProb = float64(win) / 1000.0
				bestDrawProb = float64(draw) / 1000.0
				bestLossProb = float64(loss) / 1000.0
			}
		}
		if strings.HasPrefix(response, "bestmove") {
			parts := strings.Fields(response)
			if len(parts) >= 2 {
				bestMove = parts[1]
			}
			break
		}
	}

	result := &AnalysisResult{
		BestMove:      bestMove,
		BestMoveScore: lastScore / 100, // Convert centipawns to pawns
	}

	// If the chosen move is different from the best move, evaluate it
	if bestMove != lastMove {
		// Set up position before the last move again
		if len(moves) > 1 {
			e.sendCommand(fmt.Sprintf("position startpos moves %s", strings.Join(moves[:len(moves)-1], " ")))
		} else {
			e.sendCommand("position startpos")
		}

		// Evaluate the specific last move using searchmoves
		e.sendCommand(fmt.Sprintf("go depth %d searchmoves %s", depth, lastMove))

		for response := range e.responses {
			if strings.Contains(response, "score cp ") {
				// Parse score
				parts := strings.Split(response, "score cp ")
				if len(parts) > 1 {
					fmt.Sscanf(parts[1], "%f", &result.Score)
					result.Score = result.Score / 100 // Convert centipawns to pawns
				}
			}
			if strings.Contains(response, " wdl ") {
				// Parse WDL statistics (win/draw/loss in permille)
				parts := strings.Split(response, " wdl ")
				if len(parts) > 1 {
					var win, draw, loss int
					fmt.Sscanf(parts[1], "%d %d %d", &win, &draw, &loss)
					result.WinProb = float64(win) / 1000.0
					result.DrawProb = float64(draw) / 1000.0
					result.LossProb = float64(loss) / 1000.0
				}
			}
			if strings.HasPrefix(response, "bestmove") {
				break
			}
		}
	} else {
		// If the chosen move is the best move, use the same score and WDL statistics
		result.Score = result.BestMoveScore
		result.WinProb = bestWinProb
		result.DrawProb = bestDrawProb
		result.LossProb = bestLossProb
	}

	return result, nil
}

// Close shuts down the Stockfish engine
func (e *StockfishEngine) Close() error {
	e.sendCommand("quit")
	return e.cmd.Wait()
}
