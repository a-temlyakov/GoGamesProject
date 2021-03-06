package zombie_dice

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Akavall/GoGamesProject/dice"
)

//TO-DO: should probably define these to be configurable for each new game...
const WINNING_SCORE = 13
const DICE_TO_DEAL = 3
const SHOTS_UNTIL_DEAD = 3

type GameState struct {
	Players
	ZombieDeck
	PlayerTurn int
	Winner     Player
	GameOver   bool
	IsActive   bool
}

type Players []Player

type Player struct {
	*PlayerState
	Name       string
	IsAI       bool
	TotalScore *int
}

type PlayerState struct {
	TurnsTaken   int
	CurrentScore int
	TimesShot    int
	BrainsRolled int
	WalksTaken   int
	IsDead       bool
}

func (gs *GameState) EndTurn() {
	next_player_turn := gs.PlayerTurn + 1

	if next_player_turn >= len(gs.Players) {
		next_player_turn = 0
		gs.endRound()
	}

	gs.PlayerTurn = next_player_turn

	deck := InitZombieDeck()
	deck.Shuffle()
	gs.ZombieDeck = deck
}

func (gs *GameState) endRound() {
	//check scores
	//TO-DO: need to handle ties
	max_score := 0
	var player_with_max Player
	for _, p := range gs.Players {
		if *p.TotalScore >= max_score {
			max_score = *p.TotalScore
			player_with_max = p
		}
	}

	if max_score >= WINNING_SCORE {
		gs.Winner = player_with_max
		gs.GameOver = true
	}
}

func (ps *PlayerState) Reset() {
	ps.TurnsTaken = 0
	ps.CurrentScore = 0
	ps.TimesShot = 0
	ps.IsDead = false
}

func InitGameState(players Players) (gs GameState, err error) {
	deck := InitZombieDeck()
	deck.Shuffle()

	return GameState{Players: players, ZombieDeck: deck, PlayerTurn: 0, Winner: Player{}, GameOver: false, IsActive: false}, nil
}

func (p *Player) TakeTurn(deck *ZombieDeck) (s string, err error) {
	if p.PlayerState.IsDead == true {
		return "", errors.New(fmt.Sprintf("Player %s is dead and cannot take more turns!", p.Name))
	}

	dices_to_roll, err := deck.DealDice(DICE_TO_DEAL)
	if err != nil {
		return
	}

	turn_result := ""
	sides := make([]dice.Side, 0)
	for _, d := range dices_to_roll {
		side := d.Roll()
		sides = append(sides, side)
		turn_result += d.Name + "," + side.Name + ";" //poor way to do this, but will do for now
		log.Printf("%s rolled: %s, %s\n", p.Name, d.Name, side.Name)

		if side.Name == "brain" {
			p.PlayerState.CurrentScore++
			p.PlayerState.BrainsRolled++
		} else if side.Name == "shot" {
			p.PlayerState.TimesShot++
		} else if side.Name == "walk" {
			// Since walks get replayed we have to
			// put them back in the deck
			deck.AddDice(d)
			p.PlayerState.WalksTaken++
		} else {
			return turn_result, errors.New(fmt.Sprintf("Unrecognized dice side has been rolled: %s", side.Name))
		}
	}

	if p.PlayerState.TimesShot >= SHOTS_UNTIL_DEAD {
		p.PlayerState.IsDead = true
	}

	p.PlayerState.TurnsTaken++
	return turn_result, nil //TO-DO: need proper return here that significies dice color + side rolled
}

func InitPlayerState() *PlayerState {
	return &PlayerState{TurnsTaken: 0, CurrentScore: 0, TimesShot: 0, IsDead: false}
}

func shouldKeepGoing(p Player, deck *ZombieDeck) bool {
	if !p.IsAI {
		log.Println("Do you want to continue? Hit 1 to continue and 0 to stop")
	} else {
		time.Sleep(2 * 1e9)
	}

	var answer int

	switch p.Name {
	case "human":
		answer = get_terminal_input()
	case "greedy":
		answer = GreedyAI(p.PlayerState.TimesShot)
	case "careful":
		answer = CarefulAI(p.PlayerState.TimesShot)
	case "random":
		answer = RandomAI()
	case "simulationist":
		answer = SimulationistAI(p.PlayerState.TimesShot, p.PlayerState.BrainsRolled, p.PlayerState.WalksTaken, deck)
	}

	if answer == 0 {
		if p.IsAI {
			log.Println("turn ending...")
			time.Sleep(2 * 1e9)
		}
		return false
	}
	return true
}

func PlayWithAI() {
	ai_name := select_ai()

	players := make([]Player, 2)

	t1 := 0
	t2 := 0
	players[0] = Player{PlayerState: InitPlayerState(), Name: "human", IsAI: false, TotalScore: &t1}
	players[1] = Player{PlayerState: InitPlayerState(), Name: ai_name, IsAI: true, TotalScore: &t2}

	gameState, err := InitGameState(players)

	if err != nil {
		log.Printf("Error occured while initializing game state")
	}

	for {
		//Using explicit for loop here because need to change state
		//range returns a copy, so state is lost after each iteration
		for i := 0; i < len(gameState.Players); i++ {
			p := gameState.Players[i]
			log.Printf("Player %s is taking turn; Players total score: %d", p.Name, *p.TotalScore)
			for {
				_, err := p.TakeTurn(&gameState.ZombieDeck)

				if err != nil {
					log.Printf("Error occured while player %s was taking turn: %s", p.Name, err.Error())
					break
				}

				log.Printf("Current score: %d; Times shot: %d", p.PlayerState.CurrentScore, p.PlayerState.TimesShot)

				if p.PlayerState.IsDead {
					log.Printf("Player %s has died! No points scored.", p.Name)
					p.PlayerState.Reset()
					time.Sleep(3 * 1e9)
					break
				}

				if !shouldKeepGoing(p, &gameState.ZombieDeck) {
					log.Printf("Player %s chose to stop, added %d to total score", p.Name, p.PlayerState.CurrentScore)
					*p.TotalScore += p.PlayerState.CurrentScore
					log.Printf("Player %s total score is now: %d", p.Name, *p.TotalScore)

					//p.PlayerState = InitPlayerState()
					p.PlayerState.Reset()
					break
				}
			}
			gameState.EndTurn()
		}

		if gameState.GameOver == true {
			log.Printf("Player %s won!", gameState.Winner.Name)
			break
		}
	}
}

func get_terminal_input() int {
	reader := bufio.NewReader(os.Stdin)
	raw_string, _ := reader.ReadString('\n')
	clean_string := strings.Replace(raw_string, "\n", "", -1)
	answer, _ := strconv.Atoi(clean_string)
	return answer
}

func select_ai() string {
back:
	fmt.Println("Please Select an AI you want to play against")
	fmt.Printf("Greedy : press %d\n", 1)
	fmt.Printf("Careful : press %d\n", 2)
	fmt.Printf("Random : press %d\n", 3)
	fmt.Printf("Simulationist : press %d\n", 4)

	answer := get_terminal_input()
	switch answer {
	case 1:
		return "greedy"
	case 2:
		return "careful"
	case 3:
		return "random"
	case 4:
		return "simulationist"
	default:
		fmt.Println("This is not a valid selection, please try again")
		goto back
	}
}
