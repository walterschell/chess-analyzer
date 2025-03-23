package chessanalysis

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	chess "github.com/corentings/chess/v2"
)

type MoveClassification int

const (
	Neutral MoveClassification = iota
	Blunder
	Questionable
	Good
	Excellent
	Winning
)

func (c MoveClassification) String() string {
	return []string{"Neutral", "Blunder", "Questionable", "Good", "Excellent", "Winning"}[c]
}

type MoveAnalysis struct {
	MoveNumber                   int
	Color                        string
	MoveText                     string
	Score                        float64
	CentipawnDifference          float64
	WinningProbability           float64
	WinningProbabilityDifference float64
	Classification               MoveClassification
	IsBestMove                   bool
	BestMove                     string
	BestMoveSAN                  string
	BestMoveScore                float64
}

func (m *MoveAnalysis) String() string {
	return fmt.Sprintf("Move %d: %s (Score: %.2f, Centipawn Difference: %.2f, Classification: %s, Is Best Move: %t)",
		m.MoveNumber, m.MoveText, m.Score, m.CentipawnDifference, m.Classification, m.IsBestMove)
}

// Chess annotation symbols for move classifications
var classificationAnnotations = map[MoveClassification]string{
	Blunder:      "??",
	Questionable: "?",
	Neutral:      "",
	Good:         "!",
	Excellent:    "!!",
	Winning:      "⩲",
}

// MoveAnalysisJSON is the JSON representation of MoveAnalysis
type moveAnalysisJSON struct {
	MoveNumber                   int     `json:"moveNumber"`
	Color                        string  `json:"color"`
	MoveText                     string  `json:"moveText"`
	Score                        float64 `json:"score"`
	CentipawnDifference          float64 `json:"centipawnDifference"`
	WinningProbability           float64 `json:"winningProbability"`
	WinningProbabilityDifference float64 `json:"winningProbabilityDifference"`
	Classification               string  `json:"classification"`       // Human readable
	ClassificationSymbol         string  `json:"classificationSymbol"` // Chess annotation
	IsBestMove                   bool    `json:"isBestMove"`
	BestMove                     string  `json:"bestMove"`
	BestMoveSAN                  string  `json:"bestMoveSAN"`
	BestMoveScore                float64 `json:"bestMoveScore"`
}

// MarshalJSON implements custom JSON serialization for MoveAnalysis
func (m *MoveAnalysis) MarshalJSON() ([]byte, error) {
	return json.Marshal(moveAnalysisJSON{
		MoveNumber:                   m.MoveNumber,
		Color:                        m.Color,
		MoveText:                     m.MoveText,
		Score:                        m.Score,
		CentipawnDifference:          m.CentipawnDifference,
		WinningProbability:           m.WinningProbability,
		WinningProbabilityDifference: m.WinningProbabilityDifference,
		Classification:               m.Classification.String(),
		ClassificationSymbol:         classificationAnnotations[m.Classification],
		IsBestMove:                   m.IsBestMove,
		BestMove:                     m.BestMove,
		BestMoveSAN:                  m.BestMoveSAN,
		BestMoveScore:                m.BestMoveScore,
	})
}

// classifyMove determines the quality of a move based on WDL probabilities
func classifyMove(winProb, bestWinProb float64) MoveClassification {
	// Calculate the difference in winning probability
	winProbDiff := winProb - bestWinProb

	switch {
	case winProbDiff <= -0.2: // More than 20% worse than best move
		return Blunder
	case winProbDiff <= -0.1: // More than 10% worse than best move
		return Questionable
	case winProbDiff >= 0.1: // More than 10% better than best move
		return Excellent
	case winProbDiff >= 0.05: // More than 5% better than best move
		return Good
	case winProb >= 0.95: // Almost certain win
		return Winning
	default:
		return Neutral
	}
}

func moveToSan(startingPosition *chess.Position, move *chess.Move) string {
	return chess.AlgebraicNotation{}.Encode(startingPosition, move)
}

func moveToUci(startingPosition *chess.Position, move *chess.Move) string {
	return chess.UCINotation{}.Encode(startingPosition, move)
}

// calculateWinningProbability converts centipawn score to winning probability
// using a logistic function
func calculateWinningProbability(score float64) float64 {
	return 1.0 / (1.0 + math.Exp(-score/100.0))
}

type AnalyzeChessGameOptions struct {
	Depth int
}

var defaultAnalyzeChessGameOptions = AnalyzeChessGameOptions{
	Depth: 2,
}

type AnalyzeChessGameOption func(*AnalyzeChessGameOptions)

func WithDepth(depth int) AnalyzeChessGameOption {
	return func(opts *AnalyzeChessGameOptions) {
		opts.Depth = depth
	}
}

// AnalyzeChessGameStreaming analyzes a chess game move by move, sending results through a channel
func AnalyzeChessGameStreaming(pgn string, opts ...AnalyzeChessGameOption) (<-chan *MoveAnalysis, <-chan error) {
	// Process options
	analysisOpts := defaultAnalyzeChessGameOptions
	for _, opt := range opts {
		opt(&analysisOpts)
	}

	results := make(chan *MoveAnalysis)
	errc := make(chan error, 1)

	if pgn == "" {
		errc <- fmt.Errorf("empty PGN")
		close(results)
		close(errc)
		return results, errc
	}

	go func() {
		defer close(results)
		defer close(errc)

		// Initialize Stockfish engine
		log.Info("Initializing Stockfish engine")
		engine, err := NewStockfishEngine()
		if err != nil {
			errc <- fmt.Errorf("failed to initialize Stockfish: %v", err)
			return
		}
		defer engine.Close()
		log.Info("Stockfish engine initialized")

		// Parse PGN
		log.Info("Parsing PGN")
		reader := strings.NewReader(pgn)
		pgnOpt, err := chess.PGN(reader)
		if err != nil {
			log.Error("Error parsing PGN", "error", err)
			errc <- fmt.Errorf("error parsing PGN: %v", err)
			return
		}
		log.Info("PGN parsed")

		// Create new game from PGN
		log.Info("Creating new game from PGN")
		game := chess.NewGame(pgnOpt)
		log.Info("Game created", "moves", len(game.Moves()))

		moves := game.Moves()
		var previousScore float64 = StartingPositionScore
		var uciMoves []string
		runningGame := chess.NewGame()
		// Analyze each position
		for i := 0; i < len(moves); i++ {
			tempGame := runningGame.Clone()
			lastMove := moves[i]
			lastMoveSan := moveToSan(tempGame.Position(), lastMove)

			err = runningGame.PushMove(lastMoveSan, &chess.PushMoveOptions{
				ForceMainline: true,
			})
			if err != nil {
				log.Error("Error moving in running game", "error", err, "move", moves[i].String(), "san", lastMoveSan, "position", tempGame.Position().String())
				errc <- fmt.Errorf("error moving in running game: %v", err)
				return
			}

			moveNum := (i / 2) + 1
			color := "White"
			if i%2 == 1 {
				color = "Black"
				// Negate previous score for Black's perspective
				previousScore = -previousScore
			}

			// Get the current move
			moveText := lastMoveSan
			uciMoves = append(uciMoves, moveToUci(tempGame.Position(), lastMove))

			// Create analysis entry
			analysis := &MoveAnalysis{
				MoveNumber: moveNum,
				Color:      color,
				MoveText:   moveText,
			}

			// Analyze position after the move
			result, err := engine.analyzeLastMove(uciMoves, analysisOpts.Depth)
			if err != nil {
				errc <- fmt.Errorf("analysis error at move %d: %v", moveNum, err)
				return
			}
			analysis.BestMove = result.BestMove

			// Convert best move to SAN format and get its score
			if result.BestMove != "" {
				bestMove, err := chess.UCINotation{}.Decode(tempGame.Position(), result.BestMove)
				if err != nil {
					log.Error("Error parsing best move", "error", err, "bestMove", result.BestMove)
					continue
				}
				analysis.BestMoveSAN = chess.AlgebraicNotation{}.Encode(tempGame.Position(), bestMove)
			}

			// Store the score and probabilities
			analysis.Score = result.Score
			analysis.BestMoveScore = result.BestMoveScore
			analysis.WinningProbability = result.WinProb
			analysis.WinningProbabilityDifference = result.WinProb - result.BestMoveWinProb

			if color == "Black" {
				analysis.Score = -result.Score
				analysis.BestMoveScore = -result.BestMoveScore
				analysis.WinningProbability = result.LossProb // For Black, winning probability is the opponent's loss probability
				analysis.WinningProbabilityDifference = result.LossProb - result.BestMoveLossProb
			}

			// Calculate centipawn difference for backward compatibility
			analysis.CentipawnDifference = (result.BestMoveScore - result.Score) * 100

			// Classify the move based on WDL probabilities
			analysis.Classification = classifyMove(analysis.WinningProbability, result.BestMoveWinProb)
			analysis.IsBestMove = result.BestMove == moveToUci(tempGame.Position(), lastMove)

			// Send analysis result
			results <- analysis

			// Update for next iteration
			previousScore = analysis.Score
			if color == "Black" {
				previousScore = -previousScore
			}
		}
	}()

	return results, errc
}

func AnalyzeChessGame(pgn string, opts ...AnalyzeChessGameOption) ([]MoveAnalysis, error) {
	// Start streaming analysis
	movesChan, errChan := AnalyzeChessGameStreaming(pgn, opts...)

	// Collect results
	results := make([]MoveAnalysis, 0)
	for move := range movesChan {
		results = append(results, *move) // Dereference the pointer when adding to results
	}

	// Check for any errors
	if err := <-errChan; err != nil {
		return nil, err
	}

	log.Info("Analysis complete", "moves", len(results))
	return results, nil
}
