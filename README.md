# Chess Game Analyzer

A web application that allows you to analyze chess games using Stockfish. Upload your games in PGN format, visualize them move by move, and get computer analysis of each position.

## Requirements

- Go 1.21 or later
- Stockfish chess engine (must be installed and available in your PATH)

## Installation

1. Install Stockfish:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install stockfish

   # macOS with Homebrew
   brew install stockfish

   # Windows
   # Download from https://stockfishchess.org/download/ and add to PATH
   ```

2. Clone the repository:
   ```bash
   git clone https://github.com/walterschell/chess-analyzer.git
   cd chess-analyzer
   ```

3. Install dependencies:
   ```bash
   go mod vendor
   ```

## Running the Application

1. Start the server:
   ```bash
   go run webapp.go
   ```

2. Open your browser and navigate to:
   ```
   http://localhost:8080
   ```

## Usage

1. Paste your chess game in PGN format into the text area
2. Click "Load Game" to visualize the game
3. Use the control buttons to navigate through the moves
4. The analysis panel will show Stockfish's evaluation for each position

## Example PGN

Here's a sample game you can try:
```
[Event "World Championship Match"]
[Site "London ENG"]
[Date "2018.11.09"]
[Round "1"]
[White "Caruana, Fabiano"]
[Black "Carlsen, Magnus"]
[Result "1/2-1/2"]

1. e4 c5 2. Nf3 Nc6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 e5 6. Ndb5 d6 7. Nd5 Nxd5 8. exd5 Ne7 9. c4 Ng6 10. Qa4 Bd7 11. Qb4 Qb8 12. h4 h5 13. Be3 a6 14. Nc3 a5 15. Qb3 a4 16. Qd1 Be7 17. g3 Qc8 18. Be2 Bg4 19. Rc1 O-O 20. Bxg4 hxg4 21. h5 Ne7 22. Kf1 e4 23. Kg2 f5 24. h6 gxh6 25. Rxh6 Kg7 26. Rh4 Rh8 27. Rxh8 Qxh8 28. Qd4+ Kg8 29. Rh1 Bf6 30. Qd2 Qg7 31. Rh6 Rf8 32. Qd4 Qf7 33. Rh1 Ng6 34. Bd2 Ne5 35. f3 exf3+ 36. Kxf3 f4 37. gxf4 Nxc4 38. Bc1 Ne3 39. Kg2 Bc3 40. Qf2 Qg7+ 41. Kf1 Rxf4 42. Qh4 Rf8 43. Qh5 Qf6 44. Ne4 Qf4 45. Nf2 Nf5 46. Rh4 Qc1 47. Rh1 Qf4 48. Rh4 Qc1 49. Rh1 Qf4 50. Rh4 1/2-1/2
```
