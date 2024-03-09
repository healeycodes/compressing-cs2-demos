package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang/geo/r3"
	"github.com/healeycodes/compress-cs2-demo/github.com/healeycodes/compress-cs2-demo/optimal"
	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"google.golang.org/protobuf/proto"
)

func main() {
	path := "pera-vs-system5-m1-vertigo.dem"

	t := time.Now()
	naive(path)
	fmt.Printf("naive done %s\n", time.Since(t))

	t = time.Now()
	better(path)
	fmt.Printf("optimal done %s\n", time.Since(t))
}

func naive(path string) {
	type Player struct {
		Id        int
		Name      string
		Position  r3.Vector
		Equipment []string
	}

	type Frame struct {
		Players []Player
	}

	type Game struct {
		Frames []Frame
	}

	f, err := os.Open("pera-vs-system5-m1-vertigo.dem")
	if err != nil {
		log.Panic("failed to open demo file: ", err)
	}
	defer f.Close()

	p := dem.NewParser(f)
	defer p.Close()

	game := Game{
		Frames: []Frame{},
	}

	moreFrames, err := p.ParseNextFrame()
	for ; moreFrames && err == nil; moreFrames, err = p.ParseNextFrame() {

		frame := Frame{Players: []Player{}}

		for _, player := range p.GameState().Participants().Playing() {

			equipment := []string{}
			for _, equip := range player.Weapons() {
				equipment = append(equipment, equip.String())
			}

			frame.Players = append(frame.Players, Player{
				Id:        int(player.SteamID64),
				Name:      player.Name,
				Position:  player.Position(),
				Equipment: equipment,
			})
		}

		game.Frames = append(game.Frames, frame)
	}

	if err != nil {
		log.Panicf("Failed to parse: %s\n", err)
	}

	jsonData, err := json.Marshal(game)
	if err != nil {
		log.Panicf("Failed to marshal JSON: %s\n", err)
	}

	f, err = os.Create("./naive.json")
	if err != nil {
		log.Panicf("Failed to write JSON: %s\n", err)
	}
	f.Write(jsonData)
}

func better(path string) {
	type Player = struct {
		Id      uint64 `json:"id"`
		IdShort uint32 `json:"idShort"`
		Name    string `json:"name"`
	}

	type PlayerMeta map[uint64]Player
	type EquipmentMeta map[string]int32

	type Frame struct {
		PlayerSpawn     []uint32             `json:"playerSpawn,omitempty"`
		PlayerDeath     []uint32             `json:"playerDeath,omitempty"`
		PositionChange  map[uint32]r3.Vector `json:"positionChange,omitempty"`
		EquipmentChange map[uint32][]int32   `json:"equipmentChange,omitempty"`
	}

	type Game struct {
		PlayerMeta    `json:"playerMeta"`
		EquipmentMeta `json:"equipmentMeta"`
		Frames        []Frame `json:"frames"`
	}

	f, err := os.Open(path)
	if err != nil {
		log.Panicf("Failed to open demo file: %s\n", err)
	}
	defer f.Close()

	p := dem.NewParser(f)
	defer p.Close()

	game := Game{
		PlayerMeta:    make(PlayerMeta),
		EquipmentMeta: make(EquipmentMeta),
		Frames:        make([]Frame, 0),
	}

	type FrameInfo struct {
		Position  r3.Vector
		Equipment []int32
	}

	var playerIds uint32 = 1
	var equipIds int32 = 1

	var curFrame map[uint32]FrameInfo
	lastFrame := map[uint32]FrameInfo{}

	moreFrames, err := p.ParseNextFrame()
	for ; moreFrames && err == nil; moreFrames, err = p.ParseNextFrame() {
		curFrame = map[uint32]FrameInfo{}

		for _, player := range p.GameState().Participants().Playing() {
			var idShort uint32
			if meta, ok := game.PlayerMeta[player.SteamID64]; !ok {
				idShort = playerIds
				playerIds++
				game.PlayerMeta[player.SteamID64] = Player{
					Id:      player.SteamID64,
					IdShort: idShort,
					Name:    player.Name,
				}
			} else {
				idShort = meta.IdShort
			}

			equipment := []int32{}
			for _, equip := range player.Weapons() {
				if idEquip, ok := game.EquipmentMeta[equip.String()]; !ok {
					idEquip = equipIds
					equipIds++
					game.EquipmentMeta[equip.String()] = idEquip
					equipment = append(equipment, idEquip)
				} else {
					equipment = append(equipment, idEquip)
				}
			}

			curFrame[idShort] = FrameInfo{
				Position:  player.Position(),
				Equipment: equipment,
			}
		}
		frame := Frame{
			PlayerSpawn:     []uint32{},
			PlayerDeath:     []uint32{},
			PositionChange:  map[uint32]r3.Vector{},
			EquipmentChange: map[uint32][]int32{},
		}

		for id := range lastFrame {
			if _, ok := curFrame[id]; !ok {
				frame.PlayerDeath = append(frame.PlayerDeath, id)
			}
		}

		for id, info := range curFrame {
			if lastInfo, ok := lastFrame[id]; !ok {
				frame.PlayerSpawn = append(frame.PlayerSpawn, id)
				frame.PositionChange[id] = info.Position
				frame.EquipmentChange[id] = info.Equipment
			} else {
				if lastInfo.Position.Cmp(info.Position) != 0 {
					frame.PositionChange[id] = info.Position
				}

				// Add equipment
				for _, idEquip := range info.Equipment {
					if !contains[int32](lastInfo.Equipment, idEquip) {
						if frame.EquipmentChange[id] == nil {
							frame.EquipmentChange[id] = []int32{}
						}
						frame.EquipmentChange[id] = append(frame.EquipmentChange[id], idEquip)
					}
				}

				// Remove equipment
				for _, idEquip := range lastInfo.Equipment {
					if contains[int32](lastInfo.Equipment, idEquip) && !contains[int32](info.Equipment, idEquip) {
						if frame.EquipmentChange[id] == nil {
							frame.EquipmentChange[id] = []int32{}
						}
						frame.EquipmentChange[id] = append(frame.EquipmentChange[id], idEquip)
					}
				}
			}
		}

		lastFrame = curFrame
		game.Frames = append(game.Frames, frame)
	}

	if err != nil {
		log.Panicf("Failed to parse: %s\n", err)
	}

	jsonData, err := json.Marshal(game)
	if err != nil {
		log.Panicf("Failed to marshal JSON: %s\n", err)
	}

	f, err = os.Create("./better.json")
	if err != nil {
		log.Panicf("Failed to write JSON: %s\n", err)
	}
	f.Write(jsonData)

	protoGame := &optimal.Game{
		PlayerMeta: &optimal.PlayerMeta{
			Players: map[uint64]*optimal.Player{},
		},
		EquipmentMeta: &optimal.EquipmentMeta{
			Equipment: map[string]int32{},
		},
		Frames: []*optimal.Frame{},
	}

	for id, meta := range game.PlayerMeta {
		protoGame.PlayerMeta.Players[id] = &optimal.Player{
			Id:      meta.Id,
			IdShort: uint32(meta.IdShort),
			Name:    meta.Name,
		}
	}

	for id, equip := range game.EquipmentMeta {
		protoGame.EquipmentMeta.Equipment[id] = int32(equip)
	}

	for _, frame := range game.Frames {
		protoGame.Frames = append(protoGame.Frames, &optimal.Frame{
			PlayerSpawn:     frame.PlayerSpawn,
			PlayerDeath:     frame.PlayerDeath,
			PositionChange:  convertVectorMap(frame.PositionChange),
			EquipmentChange: convertEquipmentList(frame.EquipmentChange),
		})
	}

	data, err := proto.Marshal(protoGame)
	if err != nil {
		log.Fatalf("Failed to marshal protobuf: %v", err)
	}

	if err := os.WriteFile("better.proto", data, 0644); err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}
}

func convertVectorMap(m map[uint32]r3.Vector) map[uint32]*optimal.Vector {
	m2 := map[uint32]*optimal.Vector{}
	for id, v := range m {
		m2[id] = &optimal.Vector{
			X: v.X,
			Y: v.Y,
			Z: v.Z,
		}
	}
	return m2
}

func convertEquipmentList(l map[uint32][]int32) map[uint32]*optimal.EquipmentList {
	el := map[uint32]*optimal.EquipmentList{}
	for k, v := range l {
		el[k] = &optimal.EquipmentList{Equipment: v}
	}
	return el
}

func contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}
