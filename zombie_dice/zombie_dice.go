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
	"github.com/nu7hatch/gouuid"
)

type GameState struct {
	Players
	ZombieDeck
	uuid         *uuid.UUID
	round_number int
}

type Players []Player

type Player struct {
	playerState
	Name        string
	is_ai       bool
	total_score *int
}

type playerState struct {
	turns_taken   int
	current_score int
	times_shot    int
	is_dead       bool
}

func InitGameState(players Players) (gs GameState, err error) {
	deck := InitZombieDeck()
	uuid, err := uuid.NewV4()
	if err != nil {
		return
	}

	return GameState{Players: players, ZombieDeck: deck, uuid: uuid, round_number: 0}, nil
}

func (p *Player) TakeTurn(deck *ZombieDeck) (s dice.Sides, err error) {
	dices_to_roll, err := deck.DealDice(3)
	if err != nil {
		return
	}

	sides := make([]dice.Side, 0)
	for _, d := range dices_to_roll {
		side := d.Roll()
		sides = append(sides, side)
		log.Printf("%s rolled: %s, %s\n", p.Name, d.Name, side.Name)

		if side.Name == "brain" {
			p.playerState.current_score++
		} else if side.Name == "shot" {
			p.playerState.times_shot++
		} else if side.Name == "walk" {
			// Since walks get replayed we have to
			// put them back in the deck
			deck.AddDice(d)
		} else {
			return nil, errors.New(fmt.Sprintf("Unrecognized dice side has been rolled: %s", side.Name))
		}
	}

	if p.playerState.times_shot >= 3 {
		p.playerState.is_dead = true
		log.Printf("%s has been shot three or more times \n", p.Name)
	}

	p.playerState.turns_taken++
	return sides, nil
}

func initPlayerState() (ps playerState) {
	return playerState{turns_taken: 0, current_score: 0, times_shot: 0, is_dead: false}
}

func shouldKeepGoing(p Player) bool {
	if p.Name == "human" {
		log.Println("Do you want to continue? Hit 1 to continue and 0 to stop")
	}

	if p.Name != "human" {
		time.Sleep(5 * 1e9)
	}

	var answer int

	switch p.Name {
	case "human":
		answer = get_terminal_input()
	case "greedy":
		answer = GreedyAI(p.playerState.times_shot)
	case "careful":
		answer = CarefulAI(p.playerState.times_shot)
	case "random":
		answer = RandomAI()

	}

	if answer == 0 {
		if p.Name != "human" {
			log.Println("turn ending...")
			time.Sleep(3 * 1e9)
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
	players[0] = Player{playerState: initPlayerState(), Name: "human", is_ai: false, total_score: &t1}
	players[1] = Player{playerState: initPlayerState(), Name: ai_name, is_ai: true, total_score: &t2}

	gameState, err := InitGameState(players)

	if err != nil {
		log.Printf("Error occured while initializing game state")
	}

	for {
		for _, p := range gameState.Players {
			gameState.ZombieDeck = InitZombieDeck()
			gameState.ZombieDeck.Shuffle()
		    gameState.round_number++
			log.Printf("Player %s is taking turn; Players total score: %d", p.Name, *p.total_score)
			for {
				_, err := p.TakeTurn(&gameState.ZombieDeck)

				if err != nil {
					log.Printf("Error occured while player %s was taking turn")
					break
				}

				log.Printf("Current score: %d; Times shot: %d", p.playerState.current_score, p.playerState.times_shot)

				if p.playerState.is_dead {
					log.Printf("Player %s has died! No points scored.", p.Name)
					time.Sleep(3 * 1e9)
					break
				}

				if !shouldKeepGoing(p) {
					log.Printf("Player %s chose to stop, added %d to total score", p.Name, p.playerState.current_score)
					*p.total_score += p.playerState.current_score
					log.Printf("Player %s total score is now: %d", p.Name, *p.total_score)
					p.playerState = initPlayerState()
					break
				}
			}
		}

		//TO-DO: make this a method of GameState
		//TO-DO: need to handle ties
		max_score := 0
		var player_with_max Player
		for _, p := range gameState.Players {
			if *p.total_score >= max_score {
				max_score = *p.total_score
				player_with_max = p
			}
		}

		if max_score >= 13 {
			log.Printf("Player %s won!", player_with_max.Name)
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

	answer := get_terminal_input()
	switch answer {
	case 1:
		return "greedy"
	case 2:
		return "careful"
	case 3:
		return "random"
	default:
		fmt.Println("This is not a valid selction, please try again")
		goto back
	}
}
