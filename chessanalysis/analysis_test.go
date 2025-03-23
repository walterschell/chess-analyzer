package chessanalysis

import (
	"testing"
)

const pgn = `
[Event "Live Chess"]
[Site "Chess.com"]
[Date "2025.03.20"]
[Round "-"]
[White "Player 1"]
[Black "Player 2"]
[Result "1-0"]
[CurrentPosition "2kr1bnr/ppp2ppp/1N1pq3/4p2b/2B1P3/3PBP2/PPP2P1P/R2Q2RK b - -"]
[Timezone "UTC"]
[ECO "B00"]
[ECOUrl "https://www.chess.com/openings/Nimzowitsch-Defense"]
[UTCDate "2025.03.20"]
[UTCTime "01:19:16"]
[WhiteElo "543"]
[BlackElo "498"]
[TimeControl "600"]
[Termination "Player 1 won by resignation"]
[StartTime "01:19:16"]
[EndDate "2025.03.20"]
[EndTime "01:24:15"]


1. e4 Nc6 2. Bc4 e5 3. Nf3 d6 4. Nc3 Bg4 5. O-O Nd4 $6 6. d3 $9 Nxf3+ 7. gxf3 Bh5
8. Be3 $6 Qf6 9. Nd5 Qg6+ $2 10. Kh1 O-O-O $6 11. Rg1 $1 Qe6 12. Nb6+ $3 1-0
`

const invalidPgn = `
[Event "Invalid Game"]
1. e4 e5 2. invalid_move
`

func TestAnalyzeChessGame(t *testing.T) {
	t.Log("Analyzing game...")
	results, err := AnalyzeChessGame(pgn, WithDepth(2))
	if err != nil {
		t.Fatalf("failed to analyze game: %v", err)
	}
	t.Logf("Analysis complete. Found %d moves.", len(results))

	for _, result := range results {
		t.Logf((&result).String())
	}
}

func TestAnalyzeChessGameStreaming(t *testing.T) {
	t.Run("Valid PGN", func(t *testing.T) {
		movesChan, errChan := AnalyzeChessGameStreaming(pgn, WithDepth(2))

		moveCount := 0
		var lastMove *MoveAnalysis

		// Collect moves as they come in
		for move := range movesChan {
			if move == nil {
				t.Error("received nil move analysis")
				continue
			}
			moveCount++
			lastMove = move
			t.Logf("Received move analysis: %s", move.String())
		}

		// Check for errors
		if err := <-errChan; err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify we got all moves
		if moveCount == 0 {
			t.Error("no moves were analyzed")
		}

		// Verify last move details
		if lastMove != nil {
			if lastMove.MoveText != "Nb6+" {
				t.Errorf("expected last move to be Nb6+, got %s", lastMove.MoveText)
			}
			if lastMove.Color != "White" {
				t.Errorf("expected last move color to be White, got %s", lastMove.Color)
			}
		}

		// Verify move properties
		for move := range movesChan {
			if move.MoveNumber < 1 {
				t.Errorf("invalid move number: %d", move.MoveNumber)
			}
			if move.Color != "White" && move.Color != "Black" {
				t.Errorf("invalid color: %s", move.Color)
			}
			if move.MoveText == "" {
				t.Error("empty move text")
			}
		}
	})

	// TODO: Uncomment this test once pgn parsing validateion is fixed
	// t.Run("Invalid PGN", func(t *testing.T) {
	// 	movesChan, errChan := AnalyzeChessGameStreaming(invalidPgn, WithDepth(2))

	// 	// Should receive no moves
	// 	moveCount := 0
	// 	for range movesChan {
	// 		moveCount++
	// 	}

	// 	if moveCount > 0 {
	// 		t.Errorf("expected no moves for invalid PGN, got %d", moveCount)
	// 	}

	// 	// Should receive an error
	// 	if err := <-errChan; err == nil {
	// 		t.Error("expected error for invalid PGN, got nil")
	// 	}
	// })

	t.Run("Empty PGN", func(t *testing.T) {
		movesChan, errChan := AnalyzeChessGameStreaming("", WithDepth(2))

		// Should receive no moves
		moveCount := 0
		for range movesChan {
			moveCount++
		}

		if moveCount > 0 {
			t.Errorf("expected no moves for empty PGN, got %d", moveCount)
		}

		// Should receive an error
		if err := <-errChan; err == nil {
			t.Error("expected error for empty PGN, got nil")
		}
	})
}

// 	t.Run("With Different Depths", func(t *testing.T) {
// 		depths := []int{5, 10, 15}
// 		for _, depth := range depths {
// 			t.Run(fmt.Sprintf("Depth_%d", depth), func(t *testing.T) {
// 				movesChan, errChan := AnalyzeChessGameStreaming(pgn, WithDepth(depth))

// 				moveCount := 0
// 				for move := range movesChan {
// 					if move == nil {
// 						t.Error("received nil move analysis")
// 						continue
// 					}
// 					moveCount++
// 				}

// 				if err := <-errChan; err != nil {
// 					t.Errorf("unexpected error at depth %d: %v", depth, err)
// 				}

// 				if moveCount == 0 {
// 					t.Errorf("no moves analyzed at depth %d", depth)
// 				}
// 			})
// 		}
// 	})
// }
