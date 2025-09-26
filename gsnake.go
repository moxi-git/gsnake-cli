package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

type Point struct {
	x, y int
}

type Snake struct {
	body      []Point
	direction Point
}

type Game struct {
	width, height int
	snake         Snake
	food          Point
	score         int
	gameOver      bool
	quit          bool
}

type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]uint8
	Ispeed uint32
	Ospeed uint32
}

func tcgetattr(fd int) *termios {
	var t termios
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(0x5401), uintptr(unsafe.Pointer(&t)))
	return &t
}

func tcsetattr(fd int, t *termios) {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(0x5402), uintptr(unsafe.Pointer(t)))
}

var originalTermios *termios

func enableRawMode() {
	originalTermios = tcgetattr(0)
	raw := *originalTermios
	raw.Lflag &^= 0x00000002 | 0x00000008
	raw.Cc[6] = 1
	raw.Cc[5] = 0
	tcsetattr(0, &raw)
}

func disableRawMode() {
	if originalTermios != nil {
		tcsetattr(0, originalTermios)
	}
}

func (g *Game) init() {
	g.width = 40
	g.height = 20
	g.snake = Snake{
		body: []Point{{10, 10}, {9, 10}, {8, 10}},
		direction: Point{1, 0},
	}
	g.spawnFood()
	g.score = 0
	g.gameOver = false
	g.quit = false
}

func (g *Game) spawnFood() {
	for {
		g.food = Point{
			x: rand.Intn(g.width-2) + 1,
			y: rand.Intn(g.height-2) + 1,
		}
		valid := true
		for _, segment := range g.snake.body {
			if segment.x == g.food.x && segment.y == g.food.y {
				valid = false
				break
			}
		}
		if valid {
			break
		}
	}
}

func (g *Game) update() {
	if g.gameOver || g.quit {
		return
	}

	head := g.snake.body[0]
	newHead := Point{
		x: head.x + g.snake.direction.x,
		y: head.y + g.snake.direction.y,
	}

	if newHead.x <= 0 || newHead.x >= g.width-1 || newHead.y <= 0 || newHead.y >= g.height-1 {
		g.gameOver = true
		return
	}

	for _, segment := range g.snake.body {
		if newHead.x == segment.x && newHead.y == segment.y {
			g.gameOver = true
			return
		}
	}

	g.snake.body = append([]Point{newHead}, g.snake.body...)

	if newHead.x == g.food.x && newHead.y == g.food.y {
		g.score += 1
		g.spawnFood()
	} else {
		g.snake.body = g.snake.body[:len(g.snake.body)-1]
	}
}

func (g *Game) render() {
	fmt.Print("\033[H\033[2J")
	
	board := make([][]rune, g.height)
	for i := range board {
		board[i] = make([]rune, g.width)
		for j := range board[i] {
			if i == 0 || i == g.height-1 || j == 0 || j == g.width-1 {
				board[i][j] = '█'
			} else {
				board[i][j] = ' '
			}
		}
	}

	board[g.food.y][g.food.x] = '♦'

	for _, segment := range g.snake.body {
		board[segment.y][segment.x] = '■'
	}

	fmt.Printf("Score: %d | Arrow Keys to Move | Q to Quit\n", g.score)
	for _, row := range board {
		fmt.Println(string(row))
	}
	
	if g.gameOver {
		fmt.Println("\nGAME OVER! Final Score:", g.score)
		fmt.Println("Press Q to quit or R to restart")
	}
}

func (g *Game) changeDirection(dx, dy int) {
	if g.snake.direction.x == -dx && g.snake.direction.y == -dy {
		return
	}
	g.snake.direction = Point{dx, dy}
}

func (g *Game) handleInput() {
	buffer := make([]byte, 1)
	for !g.quit {
		n, err := os.Stdin.Read(buffer)
		if err != nil || n == 0 {
			continue
		}
		
		key := buffer[0]
		
		if key == 27 {
			seq := make([]byte, 2)
			os.Stdin.Read(seq)
			if seq[0] == 91 {
				switch seq[1] {
				case 65:
					g.changeDirection(0, -1)
				case 66:
					g.changeDirection(0, 1)
				case 67:
					g.changeDirection(1, 0)
				case 68:
					g.changeDirection(-1, 0)
				}
			}
		} else {
			switch key {
			case 'q', 'Q':
				g.quit = true
				return
			case 'r', 'R':
				if g.gameOver {
					g.init()
				}
			}
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	enableRawMode()
	defer disableRawMode()
	
	game := Game{}
	game.init()
	
	go game.handleInput()
	
	go func() {
		<-c
		disableRawMode()
		fmt.Println("\nGame terminated!")
		os.Exit(0)
	}()
	
	ticker := time.NewTicker(140 * time.Millisecond)
	defer ticker.Stop()
	
	for !game.quit {
		select {
		case <-ticker.C:
			if !game.gameOver {
				game.update()
			}
			game.render()
		}
	}
	
	disableRawMode()
	fmt.Println("\nThx for playing!")
}
