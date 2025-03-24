package chessanalysis

import (
	"encoding/json"
	"fmt"
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
	Best
)

type MoveClassifier interface {
	ClassifyMove(move *MoveAnalysis) MoveClassification
}

type MoveClassifierFunc func(move *MoveAnalysis) MoveClassification

func (c MoveClassifierFunc) ClassifyMove(move *MoveAnalysis) MoveClassification {
	return c(move)
}

type ThresholdMoveClassifier struct {
	blunderWinProbThreshold       float64 // Blunder if magnitude of negative win probability delta is greater than this threshold
	blunderLossProbThreshold      float64 // Blunder if magnitude of positive loss probability delta is greater than this threshold
	questionableWinProbThreshold  float64 // Questionable if magnitude of negative win probability delta is greater than this threshold
	questionableLossProbThreshold float64 // Questionable if magnitude of positive loss probability delta is greater than this threshold
	goodWinProbThreshold          float64 // Good if magnitude of positive win probability delta is greater than this threshold
	goodLossProbThreshold         float64 // Good if magnitude of negative loss probability delta is greater than this threshold
	excellentWinProbThreshold     float64 // Excellent if magnitude of positive win probability delta is greater than this threshold
	excellentLossProbThreshold    float64
}

func (c *ThresholdMoveClassifier) ClassifyMove(move *MoveAnalysis) MoveClassification {
	if move.IsBestMove {
		return Best
	}

	var winProbDelta float64
	var lossProbDelta float64
	if move.Color == "White" {
		winProbDelta = move.WhiteWinProb - move.PreviousWhiteWinProb
		lossProbDelta = move.WhiteLossProb - move.PreviousWhiteLossProb
	} else {
		winProbDelta = move.WhiteLossProb - move.PreviousWhiteLossProb
		lossProbDelta = move.WhiteWinProb - move.PreviousWhiteWinProb
	}

	if winProbDelta <= -c.blunderWinProbThreshold || lossProbDelta >= c.blunderLossProbThreshold {
		return Blunder
	}

	if winProbDelta <= -c.questionableWinProbThreshold || lossProbDelta >= c.questionableLossProbThreshold {
		return Questionable
	}

	if winProbDelta >= c.excellentWinProbThreshold || lossProbDelta <= -c.excellentLossProbThreshold {
		return Excellent
	}

	if winProbDelta >= c.goodWinProbThreshold || lossProbDelta <= -c.goodLossProbThreshold {
		return Good
	}

	return Neutral
}

func DefaultMoveClassifier() MoveClassifier {
	return &ThresholdMoveClassifier{
		blunderWinProbThreshold:       0.2,
		blunderLossProbThreshold:      0.2,
		questionableWinProbThreshold:  0.1,
		questionableLossProbThreshold: 0.1,
		goodWinProbThreshold:          0.05,
		goodLossProbThreshold:         0.05,
		excellentWinProbThreshold:     0.1,
		excellentLossProbThreshold:    0.1,
	}
}

func (c MoveClassification) String() string {
	return []string{"Neutral", "Blunder", "Questionable", "Good", "Excellent", "Winning", "Best"}[c]
}

type MoveAnalysis struct {
	MoveNumber            int
	Color                 string
	MoveText              string
	WhiteScore            float64
	PreviousWhiteScore    float64
	IsBestMove            bool
	BestMove              string
	BestMoveSAN           string
	BestMoveWhiteScore    float64
	WhiteWinProb          float64
	WhiteDrawProb         float64
	WhiteLossProb         float64
	PreviousWhiteWinProb  float64
	PreviousWhiteDrawProb float64
	PreviousWhiteLossProb float64
	BestMoveWhiteWinProb  float64
	BestMoveWhiteDrawProb float64
	BestMoveWhiteLossProb float64
	Classification        MoveClassification
}

func (m *MoveAnalysis) String() string {
	return fmt.Sprintf("Move %d: %s (Score: %.2f, Classification: %s, Is Best Move: %t)",
		m.MoveNumber, m.MoveText, m.WhiteScore, m.Classification, m.IsBestMove)
}

// Chess annotation symbols for move classifications
var classificationAnnotations = map[MoveClassification]string{
	Blunder:      "??",
	Questionable: "?",
	Neutral:      "",
	Good:         "!",
	Excellent:    "!!",
	Winning:      "⩲",
	Best:         "*",
}

// MoveAnalysisJSON is the JSON representation of MoveAnalysis
type moveAnalysisJSON struct {
	MoveNumber            int     `json:"moveNumber"`
	Color                 string  `json:"color"`
	MoveText              string  `json:"moveText"`
	WhiteScore            float64 `json:"whiteScore"`
	PreviousWhiteScore    float64 `json:"previousWhiteScore"`
	Classification        string  `json:"classification"`       // Human readable
	ClassificationSymbol  string  `json:"classificationSymbol"` // Chess annotation
	IsBestMove            bool    `json:"isBestMove"`
	BestMove              string  `json:"bestMove"`
	BestMoveSAN           string  `json:"bestMoveSAN"`
	BestMoveWhiteScore    float64 `json:"bestMoveWhiteScore"`
	WhiteWinProb          float64 `json:"whiteWinProb"`
	WhiteDrawProb         float64 `json:"whiteDrawProb"`
	WhiteLossProb         float64 `json:"whiteLossProb"`
	BestMoveWhiteWinProb  float64 `json:"bestMoveWhiteWinProb"`
	BestMoveWhiteDrawProb float64 `json:"bestMoveWhiteDrawProb"`
	BestMoveWhiteLossProb float64 `json:"bestMoveWhiteLossProb"`
	PreviousWhiteWinProb  float64 `json:"previousWhiteWinProb"`
	PreviousWhiteDrawProb float64 `json:"previousWhiteDrawProb"`
	PreviousWhiteLossProb float64 `json:"previousWhiteLossProb"`
}

// MarshalJSON implements custom JSON serialization for MoveAnalysis
func (m *MoveAnalysis) MarshalJSON() ([]byte, error) {
	return json.Marshal(moveAnalysisJSON{
		MoveNumber:            m.MoveNumber,
		Color:                 m.Color,
		MoveText:              m.MoveText,
		WhiteScore:            m.WhiteScore,
		PreviousWhiteScore:    m.PreviousWhiteScore,
		Classification:        m.Classification.String(),
		ClassificationSymbol:  classificationAnnotations[m.Classification],
		IsBestMove:            m.IsBestMove,
		BestMove:              m.BestMove,
		BestMoveSAN:           m.BestMoveSAN,
		BestMoveWhiteScore:    m.BestMoveWhiteScore,
		WhiteWinProb:          m.WhiteWinProb,
		WhiteDrawProb:         m.WhiteDrawProb,
		WhiteLossProb:         m.WhiteLossProb,
		BestMoveWhiteWinProb:  m.BestMoveWhiteWinProb,
		BestMoveWhiteDrawProb: m.BestMoveWhiteDrawProb,
		BestMoveWhiteLossProb: m.BestMoveWhiteLossProb,
		PreviousWhiteWinProb:  m.PreviousWhiteWinProb,
		PreviousWhiteDrawProb: m.PreviousWhiteDrawProb,
		PreviousWhiteLossProb: m.PreviousWhiteLossProb,
	})
}

func moveToSan(startingPosition *chess.Position, move *chess.Move) string {
	return chess.AlgebraicNotation{}.Encode(startingPosition, move)
}

func moveToUci(startingPosition *chess.Position, move *chess.Move) string {
	return chess.UCINotation{}.Encode(startingPosition, move)
}

type AnalyzeChessGameOptions struct {
	Depth          int
	MoveClassifier MoveClassifier
}

var defaultAnalyzeChessGameOptions = AnalyzeChessGameOptions{
	Depth:          2,
	MoveClassifier: DefaultMoveClassifier(),
}

type AnalyzeChessGameOption func(*AnalyzeChessGameOptions)

func WithDepth(depth int) AnalyzeChessGameOption {
	return func(opts *AnalyzeChessGameOptions) {
		opts.Depth = depth
	}
}

func WithMoveClassifier(moveClassifier MoveClassifier) AnalyzeChessGameOption {
	return func(opts *AnalyzeChessGameOptions) {
		opts.MoveClassifier = moveClassifier
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
		var previousWhiteScore float64 = StartingPositionWhiteScore
		var previousWhiteWinProb float64 = StartingPositionWhiteWinProb
		var previousWhiteDrawProb float64 = StartingPositionWhiteDrawProb
		var previousWhiteLossProb float64 = StartingPositionWhiteLossProb
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
			}

			// Get the current move
			moveText := lastMoveSan
			uciMoves = append(uciMoves, moveToUci(tempGame.Position(), lastMove))

			// Create analysis entry
			analysis := &MoveAnalysis{
				MoveNumber:            moveNum,
				Color:                 color,
				MoveText:              moveText,
				PreviousWhiteScore:    previousWhiteScore,
				PreviousWhiteWinProb:  previousWhiteWinProb,
				PreviousWhiteDrawProb: previousWhiteDrawProb,
				PreviousWhiteLossProb: previousWhiteLossProb,
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
			analysis.WhiteScore = result.WhiteScore
			analysis.WhiteWinProb = result.WhiteWinProb
			analysis.WhiteDrawProb = result.WhiteDrawProb
			analysis.WhiteLossProb = result.WhiteLossProb
			analysis.BestMoveWhiteWinProb = result.BestMoveWhiteWinProb
			analysis.BestMoveWhiteDrawProb = result.BestMoveWhiteDrawProb
			analysis.BestMoveWhiteLossProb = result.BestMoveWhiteLossProb

			// Calculate centipawn difference for backward compatibility

			// Classify the move based on WDL probabilities
			analysis.IsBestMove = result.BestMove == moveToUci(tempGame.Position(), lastMove)

			analysis.Classification = analysisOpts.MoveClassifier.ClassifyMove(analysis)

			// Send analysis result
			results <- analysis

			// Update for next iteration
			previousWhiteScore = analysis.WhiteScore
			previousWhiteWinProb = analysis.WhiteWinProb
			previousWhiteDrawProb = analysis.WhiteDrawProb
			previousWhiteLossProb = analysis.WhiteLossProb

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
