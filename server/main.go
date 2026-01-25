package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"sync"
	"time"
    "crypto/sha1"
    "encoding/hex"
    "sort"
    "strings"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	TurnSeconds      = 50    // otomatik el değiştirme süresi 
	AutoStartSeconds = 5
	BuildPileSeconds = 15
	DiceSeconds      = 5
	DealSeconds      = 12
)

type InMsg struct {
	T     string          `json:"t"`
	ReqID string          `json:"reqId,omitempty"`
	P     json.RawMessage `json:"p,omitempty"`
}
type OutMsg struct {
	T     string      `json:"t"`
	ReqID string      `json:"reqId,omitempty"`
	P     interface{} `json:"p,omitempty"`
}
type ErrPayload struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

type Player struct {
	UserID    string `json:"userId"`
	Seat      int    `json:"seat"`      // 1..4
	Connected bool   `json:"connected"` // websocket bağlı mı?
}

type Conn struct {
	ws     *websocket.Conn
	send   chan []byte
	userID string
	roomID string


}


type GameMode string
const (
    GameModeClassic  GameMode = "CLASSIC_101"
    GameModeKatlamali GameMode = "KATLAMALI_101"
)

type PenaltyMode string
const (
    PenaltyOn  PenaltyMode = "ON"
    PenaltyOff PenaltyMode = "OFF"
)

type RoomConfig struct {
    GameMode    GameMode    `json:"gameMode"`
    PenaltyMode PenaltyMode `json:"penaltyMode"`
    HandCount   int         `json:"handCount"` // 1..11
}

type DiscardEvent struct {
    TileID string `json:"tileId"`
    Seat   int    `json:"seat"`
    UserID string `json:"userId"`
    At     int64  `json:"at"` // unix seconds (opsiyonel ama iyi)
}



type Room struct {
	ID      string          `json:"roomId"`
	State   string          `json:"state"` // LOBBY / AUTO_START / BUILD_PILES / DICE / DEALING / PLAYING
	Players map[int]*Player `json:"players"`
	OwnerID string          `json:"ownerId"`
	Updated int64           `json:"updatedAt"`

	Config        RoomConfig `json:"config"`
	ConfigLocked  bool       `json:"configLocked"`
	DealerSeat int `json:"dealerSeat"` // 1..4

	// --- Hand loop
	HandIndex int `json:"handIndex"` // 0.. (completed hands)

	// turn timer anti double-fire
	turnTimerGen int64 `json:"-"`
	// score / intermission
	IntermissionUntil int64 // unix ts, 0 = yok

	// solver cache: userId:handHash -> SolveResult
	SolverCache map[string]SolveResult





	// --- Auto start countdown
	AutoStartLeft  int         `json:"autoStartLeft"`
	autoStartTimer *time.Timer `json:"-"`

	// --- Build piles (1..15) 1'er saniye
	BuildPileIdx  int         `json:"buildPileIdx"` // 0..15 (kaçıncı deste dizildi)
	buildPileTick *time.Ticker `json:"-"`

	// --- Dice
	DiceLeft   int         `json:"diceLeft"`
	DiceValue  int         `json:"diceValue"`
	diceTicker *time.Ticker `json:"-"`
	diceStopBy string      `json:"-"` // dealer stop eden userId

	// --- Pile math
	StartPile     int    `json:"startPile"`     // 1..6
	IndicatorPile int    `json:"indicatorPile"` // 1..15
	Indicator     string `json:"indicator"`
	OkeyTileID    string `json:"okey"`

	// --- Piles
	Piles     map[int][]string `json:"-"` // 1..15 each 7 tiles
	ExtraTile string           `json:"-"` // 106. taş

	// UI/debug
	PileOwners map[int]int `json:"pileOwners"` // pileId -> seat
	PileCounts map[int]int `json:"pileCounts"` // pileId -> kaç taş kaldı (indicator çekilince vs)
	DrawPileIds []int      `json:"drawPileIds"` // dağıtılmayan 3 deste

	// --- Dealing
	DealLeft       int `json:"dealLeft"`   // 12 -> 0
	DealCursor     int `json:"dealCursor"` // hangi pile dağıtılıyor (1..15)
	DealSeatCursor int `json:"dealSeatCursor"` // sıradaki seat (dealer+1 ile başlar)
	dealTicker     *time.Ticker `json:"-"`

	// --- PLAYING (şimdilik)
	TurnSeat     int   `json:"turnSeat"`
	TurnPhase    string `json:"turnPhase"` // WAIT_DRAW / WAIT_DISCARD
	TurnDeadline int64  `json:"turnDeadline"`
	turnTimer    *time.Timer `json:"-"`

	// taş state
	DrawPile []string `json:"-"` // draw stack gerçek taş listesi (server)
	Discards []DiscardEvent `json:"-"`
	Hands    map[int][]string `json:"-"`

	// internal
	mu    sync.RWMutex     `json:"-"`
	conns map[string]*Conn `json:"-"`
}

type RoomSnapshot struct {
	RoomID  string          `json:"roomId"`
	State   string          `json:"state"`
	OwnerID string          `json:"ownerId"`
	Updated int64           `json:"updatedAt"`
	Players map[int]*Player `json:"players"`

	Config       RoomConfig `json:"config"`
	ConfigLocked bool       `json:"configLocked"`
	DealerSeat int `json:"dealerSeat"`
	IntermissionUntil int64 `json:"intermissionUntil"`

	HandIndex int `json:"handIndex"`

	AutoStartLeft int `json:"autoStartLeft"`

	BuildPileIdx int `json:"buildPileIdx"`

	DiceLeft  int `json:"diceLeft"`
	DiceValue int `json:"diceValue"`

	StartPile     int    `json:"startPile"`
	IndicatorPile int    `json:"indicatorPile"`
	Indicator     string `json:"indicator"`
	Okey          string `json:"okey"`

	PileOwners  map[int]int `json:"pileOwners"`
	PileCounts  map[int]int `json:"pileCounts"`
	DrawPileIds []int       `json:"drawPileIds"`

	DealLeft       int `json:"dealLeft"`
	DealCursor     int `json:"dealCursor"`
	DealSeatCursor int `json:"dealSeatCursor"`

	TurnSeat int    `json:"turnSeat"`
	TurnPhase string `json:"turnPhase"`
	TurnDeadline int64  `json:"turnDeadline"`

	DrawCount  int         `json:"drawCount"`
	Discards []DiscardEvent `json:"discards"`
	HandCounts map[int]int `json:"handCounts"`
	MyHand []string    `json:"myHand"`
}


type RoomPublic struct {
    RoomID       string          `json:"roomId"`
    State        string          `json:"state"`
    OwnerID      string          `json:"ownerId"`
    UpdatedAt    int64           `json:"updatedAt"`
    Players      map[int]*Player `json:"players"`
    Config       RoomConfig      `json:"config"`
    ConfigLocked bool            `json:"configLocked"`
    DealerSeat   int             `json:"dealerSeat"`
    TurnSeat     int             `json:"turnSeat"`
    TurnPhase    string          `json:"turnPhase"`
    TurnDeadline int64           `json:"turnDeadline"`
}

func handHash(hand []string) string {
	cp := append([]string(nil), hand...)
	sort.Strings(cp)
	h := sha1.Sum([]byte(strings.Join(cp, "|")))
	return hex.EncodeToString(h[:])
}


type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room

	// ✅ Global index: 1 userId aynı anda sadece 1 odada "seat" sahibi olabilir.
	// userId -> roomId
	userRoom map[string]string

	    // ✅ lobby’de duran websocketler (oda join etmemiş olabilir)
    lobbyConns map[*Conn]bool
}
// func NewRoomManager() *RoomManager {
// 	return &RoomManager{rooms: make(map[string]*Room), userRoom: make(map[string]string)}
// }

func NewRoomManager() *RoomManager {
    return &RoomManager{
        rooms: make(map[string]*Room),
        userRoom: make(map[string]string),
        lobbyConns: make(map[*Conn]bool),
    }
}


// userId başka bir odada seat sahibiyse JOIN/CREATE engellenir.
// Aynı odaya (reconnect) her zaman izin var.
func (m *RoomManager) ReserveUserRoom(userID, roomID string) error {
	if userID == "" || roomID == "" {
		return errors.New("userID/roomID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if cur, ok := m.userRoom[userID]; ok {
		if cur != roomID {
			return fmt.Errorf("user already in room %s", cur)
		}
		return nil // same room
	}

	m.userRoom[userID] = roomID
	return nil
}

func (m *RoomManager) ReleaseUserRoom(userID, roomID string) {
	if userID == "" || roomID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if cur, ok := m.userRoom[userID]; ok && cur == roomID {
		delete(m.userRoom, userID)
	}
}

func (m *RoomManager) CreateRoom(ownerUserID string, cfg RoomConfig) (*Room, error) {
	if ownerUserID == "" {
		return nil, errors.New("userId is required")
	}

	roomID, err := genRoomID(6)
	if err != nil {
		return nil, err
	}

	// ✅ atomic: userRoom + rooms aynı kilitte güncellensin
	m.mu.Lock()
	defer m.mu.Unlock()

	if cur, ok := m.userRoom[ownerUserID]; ok && cur != roomID {
		// cur != roomID her zaman true (roomID yeni) ama mantık net kalsın
		return nil, fmt.Errorf("user already in room %s", cur)
	}
	m.userRoom[ownerUserID] = roomID

	r := &Room{
		ID:         roomID,
		State:      "LOBBY",
		Players:    make(map[int]*Player),
		OwnerID:    ownerUserID,
		Updated:    time.Now().Unix(),
		DealerSeat: 1,

		Hands: make(map[int][]string, 4),
		conns: make(map[string]*Conn),

		PileOwners: make(map[int]int, 15),
		PileCounts: make(map[int]int, 15),
		SolverCache: make(map[string]SolveResult),


		Config:       cfg,
		ConfigLocked: false,
	}

	r.Players[1] = &Player{UserID: ownerUserID, Seat: 1, Connected: false}

	m.rooms[roomID] = r
	return r, nil
}


func (m *RoomManager) ListRoomsPublic() []RoomPublic {
    m.mu.RLock()
    rs := make([]*Room, 0, len(m.rooms))
    for _, r := range m.rooms { rs = append(rs, r) }
    m.mu.RUnlock()

    out := make([]RoomPublic, 0, len(rs))
    for _, r := range rs {
        r.mu.RLock()
        players := make(map[int]*Player, len(r.Players))
        for s, p := range r.Players {
            cp := *p
            players[s] = &cp
        }
        pub := RoomPublic{
            RoomID: r.ID, State: r.State, OwnerID: r.OwnerID, UpdatedAt: r.Updated,
            Players: players,
            Config: r.Config, ConfigLocked: r.ConfigLocked,
            DealerSeat: r.DealerSeat,
            TurnSeat: r.TurnSeat, TurnPhase: r.TurnPhase, TurnDeadline: r.TurnDeadline,
        }
        r.mu.RUnlock()
        out = append(out, pub)
    }
    return out
}

func (m *RoomManager) BroadcastRoomsList() {
    list := m.ListRoomsPublic()

    m.mu.RLock()
    conns := make([]*Conn, 0, len(m.lobbyConns))
    for c := range m.lobbyConns { conns = append(conns, c) }
    m.mu.RUnlock()

    b, _ := json.Marshal(OutMsg{T: "ROOMS_LIST", P: map[string]any{"rooms": list}})

    for _, c := range conns {
        select { case c.send <- b: default: }
    }
}



func (m *RoomManager) GetRoom(roomID string) (*Room, bool) {
	m.mu.RLock(); defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	return r, ok
}
var rooms = NewRoomManager()

/* =========================
   Utils
   ========================= */

func wrapSeat(s int) int {
	for s > 4 { s -= 4 }
	for s < 1 { s += 4 }
	return s
}
// func nextSeat(s int) int { return wrapSeat(s + 1) }

func nextSeat(seat int) int {
	seat++
	if seat > 4 {
		return 1
	}
	return seat
}




func wrapPile(p int) int {
	for p > 15 { p -= 15 }
	for p < 1 { p += 15 }
	return p
}

func shuffleStrings(a []string) {
	for i := len(a) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			jBig = big.NewInt(int64(i / 2))
		}
		j := int(jBig.Int64())
		a[i], a[j] = a[j], a[i]
	}
}

func calcOkeyFromIndicator(ind string) string {
	// "R07-1" -> "R08"
	if len(ind) < 4 { return "" }
	color := ind[:1]
	numStr := ind[1:3]
	n := 0
	_, _ = fmt.Sscanf(numStr, "%d", &n)
	n++
	if n > 13 { n = 1 }
	return fmt.Sprintf("%s%02d", color, n)
}

func genRoomID(n int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		x, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil { return "", err }
		out[i] = alphabet[x.Int64()]
	}
	return string(out), nil
}

func randDice1to6() int {
	x, err := rand.Int(rand.Reader, big.NewInt(6))
	if err != nil { return 1 }
	return int(x.Int64()) + 1
}

/* =========================
   Snapshot + Broadcast
   ========================= */

func (r *Room) seatOf(userID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for seat, p := range r.Players {
		if p != nil && p.UserID == userID {
			return seat
		}
	}
	return 0
}


func (r *Room) snapshotForUser(userID string) RoomSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make(map[int]*Player, 4)
	for seat, p := range r.Players {
		cp := *p
		players[seat] = &cp
	}

	// userSeat
	userSeat := 0
	for s, p := range r.Players {
		if p.UserID == userID { userSeat = s; break }
	}

	handCounts := make(map[int]int, 4)
	for seat := range r.Players {
		handCounts[seat] = len(r.Hands[seat])
	}
	myHand := []string{}
	if userSeat != 0 {
		h := r.Hands[userSeat]
		myHand = make([]string, len(h))
		copy(myHand, h)
	}

	discards := make([]DiscardEvent, len(r.Discards))
	copy(discards, r.Discards)


	po := make(map[int]int, 15)
	for k, v := range r.PileOwners { po[k] = v }
	pc := make(map[int]int, 15)
	for k, v := range r.PileCounts { pc[k] = v }
	dp := make([]int, len(r.DrawPileIds))
	copy(dp, r.DrawPileIds)

	return RoomSnapshot{
		RoomID: r.ID, State: r.State, OwnerID: r.OwnerID, Updated: r.Updated,
		Players: players,

		DealerSeat: r.DealerSeat,
		IntermissionUntil: r.IntermissionUntil,
		AutoStartLeft: r.AutoStartLeft,
		BuildPileIdx: r.BuildPileIdx,

		Config:       r.Config,
		ConfigLocked: r.ConfigLocked,
		HandIndex: r.HandIndex,


		DiceLeft: r.DiceLeft,
		DiceValue: r.DiceValue,

		StartPile: r.StartPile,
		IndicatorPile: r.IndicatorPile,
		Indicator: r.Indicator,
		Okey: r.OkeyTileID,

		PileOwners: po,
		PileCounts: pc,
		DrawPileIds: dp,

		DealLeft: r.DealLeft,
		DealCursor: r.DealCursor,
		DealSeatCursor: r.DealSeatCursor,

		TurnSeat: r.TurnSeat,
		TurnPhase: r.TurnPhase,
		TurnDeadline: r.TurnDeadline,

		DrawCount: len(r.DrawPile),
		Discards: discards,
		HandCounts: handCounts,
		MyHand: myHand,
	}
}

func (r *Room) broadcastSnapshot() {
	r.mu.RLock()
	conns := make([]*Conn, 0, len(r.conns))
	for _, c := range r.conns { conns = append(conns, c) }
	r.mu.RUnlock()

	for _, c := range conns {
		snap := r.snapshotForUser(c.userID)
		b, _ := json.Marshal(OutMsg{T: "ROOM_SNAPSHOT", P: snap})
		select {
		case c.send <- b:
		default:
		}
	}
}

func (r *Room) attachConn(userID string, c *Conn) {
	r.mu.Lock()
	r.conns[userID] = c
	for _, p := range r.Players {
		if p.UserID == userID { p.Connected = true; break }
	}
	r.Updated = time.Now().Unix()
	r.mu.Unlock()
}

func (r *Room) detachConn(userID string) {
	r.mu.Lock()
	delete(r.conns, userID)
	for _, p := range r.Players {
		if p.UserID == userID { p.Connected = false; break }
	}
	r.Updated = time.Now().Unix()

	// autoStart iptali (4 değilse)
	if r.State == "AUTO_START" || r.State == "LOBBY" {
		if len(r.Players) != 4 {
			r.stopAutoStartLocked()
			r.State = "LOBBY"
		}
	}
	r.mu.Unlock()
}

/* =========================
   Join + AutoStart
   ========================= */

func (r *Room) join(userID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// reconnect
	for s, p := range r.Players {
		if p.UserID == userID {
			p.Connected = true
			r.Updated = time.Now().Unix()
			return s, nil
		}
	}
	for seat := 1; seat <= 4; seat++ {
		if _, exists := r.Players[seat]; !exists {
			r.Players[seat] = &Player{UserID: userID, Seat: seat, Connected: true}
			r.Updated = time.Now().Unix()
			r.tryAutoStartLocked()
			return seat, nil
		}
	}
	return 0, errors.New("room is full")
}

func (r *Room) stopAutoStartLocked() {
	if r.autoStartTimer != nil {
		r.autoStartTimer.Stop()
		r.autoStartTimer = nil
	}
	r.AutoStartLeft = 0
}

func (r *Room) tryAutoStartLocked() {
	if r.State != "LOBBY" && r.State != "AUTO_START" {
		return
	}
	if len(r.Players) != 4 {
		r.stopAutoStartLocked()
		r.State = "LOBBY"
		return
	}
	// zaten çalışıyorsa
	if r.State == "AUTO_START" && r.autoStartTimer != nil {
		return
	}

	r.State = "AUTO_START"
	r.AutoStartLeft = AutoStartSeconds
	r.Updated = time.Now().Unix()

	// her saniye düşür
	r.autoStartTimer = time.AfterFunc(1*time.Second, func() {
		r.onAutoStartTick()
	})
	go r.broadcastSnapshot()
}

func (r *Room) onAutoStartTick() {
	r.mu.Lock()
	// koşullar bozulduysa iptal
	if len(r.Players) != 4 || (r.State != "AUTO_START" && r.State != "LOBBY") {
		r.stopAutoStartLocked()
		r.State = "LOBBY"
		r.Updated = time.Now().Unix()
		r.mu.Unlock()
		go r.broadcastSnapshot()
		return
	}

	if r.AutoStartLeft > 0 {
		r.AutoStartLeft--
	}
	r.Updated = time.Now().Unix()

	// bitti mi?
	if r.AutoStartLeft <= 0 {
		// autoStart timer'ı temizle
		r.stopAutoStartLocked()

		// H akışı: BUILD_PILES başlat
		r.startBuildPilesLocked()

		r.mu.Unlock()
		go r.broadcastSnapshot()
		return
	}

	// devam
	r.mu.Unlock()
	go r.broadcastSnapshot()

	r.mu.Lock()
	// tekrar kur
	r.autoStartTimer = time.AfterFunc(1*time.Second, func() {
		r.onAutoStartTick()
	})
	r.mu.Unlock()
}

func (r *Room) startIntermissionTimer() {
	time.Sleep(10 * time.Second)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Oda hâlâ intermission'da mı?
	if r.State != "INTERMISSION" {
		return
	}

	// intermission temizle
	r.IntermissionUntil = 0

	// yeni el başlat
	r.startBuildPilesLocked()
	r.Updated = time.Now().Unix()

	go r.broadcastSnapshot()
}


/* =========================
   H: Build Piles (15x7 + extra)
   ========================= */

func (r *Room) startBuildPilesLocked() {
	// precondition: r.mu LOCK altında
	r.State = "BUILD_PILES"
	r.BuildPileIdx = 0

	// reset game state
	r.Hands = make(map[int][]string, 4)
	r.Discards = nil
	r.DrawPile = nil
	r.DrawPileIds = nil

	r.StartPile = 0
	r.IndicatorPile = 0
	r.Indicator = ""
	r.OkeyTileID = ""

	// generate tiles 106
	tiles := make([]string, 0, 106)
	colors := []string{"R", "B", "G", "K"}
	for _, c := range colors {
		for n := 1; n <= 13; n++ {
			tiles = append(tiles, fmt.Sprintf("%s%02d-1", c, n))
			tiles = append(tiles, fmt.Sprintf("%s%02d-2", c, n))
		}
	}
	tiles = append(tiles, "JOKER-1", "JOKER-2")
	shuffleStrings(tiles)

	// ✅ BUILD_PILES: 1. deste 8'li, diğerleri 7'li (toplam 106)
	r.Piles = make(map[int][]string, 15)

	for i := 1; i <= 15; i++ {
		cnt := 7
		if i == 1 {
			cnt = 8
		}
		r.Piles[i] = append([]string{}, tiles[:cnt]...)
		tiles = tiles[cnt:]
	}

	// Artık extra ayrı tutulmuyor (opsiyonel: debug için boşalt)
	r.ExtraTile = ""


	// pileCounts init
	for i := 1; i <= 15; i++ { r.PileCounts[i] = 0 }

	// pileOwners (dealer rotasyonu)
	r.recalcPileOwnersLocked()

	// her saniye 1 deste "dizildi"
	// (sadece snapshot'ta BuildPileIdx artacak, piles zaten hazır)
	r.Updated = time.Now().Unix()
	go r.broadcastSnapshot()

	go func() {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()

		for range t.C {
			r.mu.Lock()
			if r.State != "BUILD_PILES" {
				r.mu.Unlock()
				return
			}
			r.BuildPileIdx++
			// counts güncelle
			for i := 1; i <= 15; i++ {
				if i <= r.BuildPileIdx {
					r.PileCounts[i] = len(r.Piles[i])
				}
			}
			r.Updated = time.Now().Unix()

			done := (r.BuildPileIdx >= 15)
			if done {
				// dice aşamasına geç
				r.startDiceLocked()
				r.mu.Unlock()
				go r.broadcastSnapshot()
				return
			}
			r.mu.Unlock()
			go r.broadcastSnapshot()
		}
	}()
}

func (r *Room) recalcPileOwnersLocked() {
	// dealerSeat -> piles 1-4
	// dealer+1 -> 5-8
	// dealer+2 -> 9-12
	// dealer+3 -> 13-15
	d := r.DealerSeat
	s2 := nextSeat(d)
	s3 := nextSeat(s2)
	s4 := nextSeat(s3)

	for p := 1; p <= 15; p++ {
		switch {
		case p >= 1 && p <= 4:
			r.PileOwners[p] = d
		case p >= 5 && p <= 8:
			r.PileOwners[p] = s2
		case p >= 9 && p <= 12:
			r.PileOwners[p] = s3
		case p >= 13 && p <= 15:
			r.PileOwners[p] = s4
		}
	}
}

/* =========================
   H: Dice (5s, dealer stop)
   ========================= */

func (r *Room) startDiceLocked() {
	r.State = "DICE"
	r.DiceLeft = DiceSeconds
	r.DiceValue = randDice1to6()
	r.diceStopBy = ""

	r.Updated = time.Now().Unix()

	go func() {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()

		for range t.C {
			r.mu.Lock()
			if r.State != "DICE" {
				r.mu.Unlock()
				return
			}

			// her saniye zar değişsin
			r.DiceValue = randDice1to6()
			if r.DiceLeft > 0 { r.DiceLeft-- }
			r.Updated = time.Now().Unix()

			// dealer stop ettiyse veya süre bitti ise
			stop := (r.DiceLeft <= 0) || (r.diceStopBy != "")
			if stop {
				// apply dice + prepare deal
				r.applyDiceAndPrepareDealLocked()

				r.mu.Unlock()
				go r.broadcastSnapshot()
				return
			}

			r.mu.Unlock()
			go r.broadcastSnapshot()
		}
	}()
}

func (r *Room) diceStop(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != "DICE" {
		return errors.New("not in DICE state")
	}
	// sadece dealer durdurabilir
	dealerUser := ""
	if p, ok := r.Players[r.DealerSeat]; ok {
		dealerUser = p.UserID
	}
	if dealerUser == "" || dealerUser != userID {
		return errors.New("only dealer can stop dice")
	}
	r.diceStopBy = userID
	r.Updated = time.Now().Unix()
	return nil
}

/* =========================
   H: Apply Dice + Prepare Deal
   ========================= */

func (r *Room) applyDiceAndPrepareDealLocked() {
	// startPile = diceValue (1..6) - kilit
	r.StartPile = r.DiceValue

	// ✅ Zar atılınca: 1. destedeki fazla taş (8'inci) StartPile'a taşınır
	// StartPile == 1 ise zaten fazla taş doğru yerde.
	if r.StartPile != 1 {
		p1 := r.Piles[1]
		if len(p1) > 0 {
			extra := p1[len(p1)-1]     // 1. destenin en üstünden al
			p1 = p1[:len(p1)-1]
			r.Piles[1] = p1

			// StartPile'ın üstüne koy
			r.Piles[r.StartPile] = append(r.Piles[r.StartPile], extra)
		}
	}


	// indicatorPile = startPile - 3 (wrap 1..15)
	r.IndicatorPile = wrapPile(r.StartPile - 3)

	// indicator = indicatorPile üst taşı (pile içinden çıkar)
	pile := r.Piles[r.IndicatorPile]
	if len(pile) > 0 {
		r.Indicator = pile[len(pile)-1]
		pile = pile[:len(pile)-1]
		r.Piles[r.IndicatorPile] = pile
	}
	r.OkeyTileID = calcOkeyFromIndicator(r.Indicator)

	// counts refresh
	for i := 1; i <= 15; i++ {
		r.PileCounts[i] = len(r.Piles[i])
	}

	// dealing hazırlığı
	r.State = "DEALING"
	r.DealLeft = DealSeconds
	r.DealCursor = r.StartPile

	// İlk deste 8'li ve dealer'ın üstüne gider => seatCursor = dealer+1
	r.DealSeatCursor = nextSeat(r.DealerSeat)

	// dağıtım ticker
	go func() {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()

		for range t.C {
			r.mu.Lock()
			if r.State != "DEALING" {
				r.mu.Unlock()
				return
			}
			if r.DealLeft <= 0 {
				// finalize
				r.finalizeAfterDealLocked()
				r.mu.Unlock()
				go r.broadcastSnapshot()
				return
			}

			r.dealOnePileLocked()

			r.Updated = time.Now().Unix()
			r.mu.Unlock()
			go r.broadcastSnapshot()
		}
	}()
}

func (r *Room) dealOnePileLocked() {
	// deal cursor 1..15 wrap
	pid := wrapPile(r.DealCursor)
	tiles := r.Piles[pid]
	r.Piles[pid] = nil



	seat := r.DealSeatCursor
	r.Hands[seat] = append(r.Hands[seat], tiles...)

	// update pileCounts
	r.PileCounts[pid] = 0

	// advance
	r.DealLeft--
	r.DealCursor = wrapPile(r.DealCursor + 1)
	r.DealSeatCursor = nextSeat(r.DealSeatCursor)
}

func (r *Room) finalizeAfterDealLocked() {
	// kalan 3 pile => draw stack (order preserved)
	// dağıtılan 12 pile startPile'dan itibaren 12 adettir
	dealt := make(map[int]bool, 15)
	cursor := r.StartPile
	for i := 0; i < DealSeconds; i++ {
		dealt[wrapPile(cursor)] = true
		cursor++
	}
	remainIds := make([]int, 0, 3)
	for p := 1; p <= 15; p++ {
		if !dealt[p] {
			remainIds = append(remainIds, p)
		}
	}
	// DrawPileIds UI
	r.DrawPileIds = remainIds

	// draw pile tiles (stack order preserved: remainIds sırasıyla ekle, üst = ilk id)
	r.DrawPile = nil
	for _, pid := range remainIds {
		r.DrawPile = append(r.DrawPile, r.Piles[pid]...)
		// pile temiz
		r.Piles[pid] = nil
		r.PileCounts[pid] = 0
	}

	// PLAYING başlangıcı
	r.State = "PLAYING"

	// ✅ Oyun dealer’ın üstünden başlar (22 taş onda)
	// dealerSeat=1 ise turnSeat=2
	r.TurnSeat = nextSeat(r.DealerSeat)
	r.TurnPhase = "WAIT_DISCARD"

	r.Updated = time.Now().Unix()



	r.ConfigLocked = true
	r.resetTurnTimerLocked()
	r.Updated = time.Now().Unix()
}


func tileSortKey(tile string) (isJoker bool, color string, num int) {
	// Joker en sona
	if len(tile) >= 5 && tile[:5] == "JOKER" {
		return true, "Z", 99
	}

	// Beklenen format: "R07-1"
	// R = color, 07 = num
	if len(tile) >= 3 {
		color = tile[:1]
		_, _ = fmt.Sscanf(tile[1:3], "%d", &num)
	}
	return false, color, num
}

func pickAutoDiscardIndex(hand []string) int {
	if len(hand) == 0 {
		return -1
	}
	best := 0
	bestJ, bestC, bestN := tileSortKey(hand[0])

	for i := 1; i < len(hand); i++ {
		j, c, n := tileSortKey(hand[i])

		// Joker her zaman daha kötü (en sona)
		if bestJ && !j {
			best = i
			bestJ, bestC, bestN = j, c, n
			continue
		}
		if j && !bestJ {
			continue
		}

		// ikisi de joker değilse: num küçük olan daha küçük
		if n < bestN {
			best = i
			bestJ, bestC, bestN = j, c, n
			continue
		}
		if n == bestN {
			// renk tie-break
			if c < bestC {
				best = i
				bestJ, bestC, bestN = j, c, n
			}
		}
	}
	return best
}


/* =========================
   TURN (şimdilik basit)
   ========================= */

func (r *Room) resetTurnTimerLocked() {
	if r.turnTimer != nil {
		r.turnTimer.Stop()
	}

	// gen++ (timer callback double-fire engeli)
	r.turnTimerGen++

	gen := r.turnTimerGen
	r.TurnDeadline = time.Now().Add(TurnSeconds * time.Second).Unix()

	r.turnTimer = time.AfterFunc(TurnSeconds*time.Second, func() {
		r.onTurnTimeout(gen)
	})
}


func (r *Room) endHandLocked() {
	// precondition: r.mu LOCK altında

	// turn timer durdur
	if r.turnTimer != nil {
		r.turnTimer.Stop()
		r.turnTimer = nil
	}
	r.TurnDeadline = 0
	r.TurnSeat = 0
	r.TurnPhase = ""

	// hand tamamlandı
	r.HandIndex++

	// oyun bitti mi?
	if r.Config.HandCount > 0 && r.HandIndex >= r.Config.HandCount {
		r.State = "FINISHED" // veya "GAME_OVER"
		r.Updated = time.Now().Unix()
		return
	}

	// yeni el: dealer +1
	r.DealerSeat = nextSeat(r.DealerSeat)

	// yeni el state sıfırla (startBuildPilesLocked zaten çoğunu resetliyor ama net olsun)
	r.AutoStartLeft = 0
	r.StartPile = 0
	r.IndicatorPile = 0
	r.Indicator = ""
	r.OkeyTileID = ""

	r.DiceLeft = 0
	r.DiceValue = 0
	r.diceStopBy = ""

	r.DealLeft = 0
	r.DealCursor = 0
	r.DealSeatCursor = 0

	r.BuildPileIdx = 0
	r.Discards = nil
	r.DrawPile = nil
	r.DrawPileIds = nil
	r.Hands = make(map[int][]string, 4)

	// pileCounts reset
	for i := 1; i <= 15; i++ {
		r.PileCounts[i] = 0
	}

	// direkt yeni el akışına gir
	// skor arası başlat (10 sn)
	r.IntermissionUntil = time.Now().Add(10 * time.Second).Unix()

	// state skor arası gibi davranır (yeni mesaj tipi eklemiyoruz)
	r.State = "INTERMISSION"

	// timer başlat
	go r.startIntermissionTimer()

	r.Updated = time.Now().Unix()
}



func (r *Room) onTurnTimeout(gen int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != "PLAYING" {
		return
	}
	// double-fire guard
	if gen != r.turnTimerGen {
		return
	}

	switch r.TurnPhase {
	case "WAIT_DRAW":
		// çekme bitti mi? => el biter
		if len(r.DrawPile) == 0 {
			r.endHandLocked()
			go r.broadcastSnapshot()
			return
		}

		// ✅ gerçek DRAW
		t := r.DrawPile[0]
		r.DrawPile = r.DrawPile[1:]
		r.Hands[r.TurnSeat] = append(r.Hands[r.TurnSeat], t)

		r.TurnPhase = "WAIT_DISCARD"
		r.Updated = time.Now().Unix()
		r.resetTurnTimerLocked()
		go r.broadcastSnapshot()
		return

	case "WAIT_DISCARD":
		hand := r.Hands[r.TurnSeat]
		if len(hand) > 0 {
			// ✅ gerçek DISCARD (en küçük taş)
			idx := pickAutoDiscardIndex(hand)
			if idx < 0 {
				idx = 0
			}
			tileID := hand[idx]
			hand[idx] = hand[len(hand)-1]
			hand = hand[:len(hand)-1]
			r.Hands[r.TurnSeat] = hand
			uid := ""
			if p, ok := r.Players[r.TurnSeat]; ok && p != nil {
				uid = p.UserID
			}
			r.Discards = append(r.Discards, DiscardEvent{
				TileID: tileID,
				Seat:   r.TurnSeat,
				UserID: uid,
				At:     time.Now().Unix(),
			})


		}

		// tur ilerlet
		r.TurnSeat = nextSeat(r.TurnSeat)
		r.TurnPhase = "WAIT_DRAW"
		r.Updated = time.Now().Unix()

		// discard sonrası çekme bitmişse -> el bitir (en temiz nokta)
		if len(r.DrawPile) == 0 {
			r.endHandLocked()
			go r.broadcastSnapshot()
			return
		}

		r.resetTurnTimerLocked()
		go r.broadcastSnapshot()
		return

	default:
		return
	}
}


/* =========================
   DRAW / DISCARD (basit)
   ========================= */

func (r *Room) draw(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != "PLAYING" { return errors.New("game not started") }
	if r.TurnPhase != "WAIT_DRAW" { return errors.New("not in WAIT_DRAW phase") }

	userSeat := 0
	for s, p := range r.Players {
		if p.UserID == userID { userSeat = s; break }
	}
	if userSeat == 0 { return errors.New("user not in room") }
	if userSeat != r.TurnSeat { return errors.New("not your turn") }
	if len(r.DrawPile) == 0 {
		// ✅ el biter -> otomatik yeni el / game over
		r.endHandLocked()
		r.Updated = time.Now().Unix()
		return nil
	}


	t := r.DrawPile[0]
	r.DrawPile = r.DrawPile[1:]
	r.Hands[userSeat] = append(r.Hands[userSeat], t)

	r.TurnPhase = "WAIT_DISCARD"
	r.resetTurnTimerLocked()
	r.Updated = time.Now().Unix()
	return nil
}

func (r *Room) discard(userID string, tileID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != "PLAYING" { return errors.New("game not started") }
	if r.TurnPhase != "WAIT_DISCARD" { return errors.New("not in WAIT_DISCARD phase") }

	userSeat := 0
	for s, p := range r.Players {
		if p.UserID == userID { userSeat = s; break }
	}
	if userSeat == 0 { return errors.New("user not in room") }
	if userSeat != r.TurnSeat { return errors.New("not your turn") }
	if tileID == "" { return errors.New("tileId required") }

	hand := r.Hands[userSeat]
	idx := -1
	for i := range hand {
		if hand[i] == tileID { idx = i; break }
	}
	if idx < 0 { return errors.New("tile not in hand") }

	hand[idx] = hand[len(hand)-1]
	hand = hand[:len(hand)-1]
	r.Hands[userSeat] = hand
	r.Discards = append(r.Discards, DiscardEvent{
		TileID: tileID,
		Seat:   userSeat,
		UserID: userID,
		At:     time.Now().Unix(),
	})


	r.TurnSeat = nextSeat(r.TurnSeat)
	r.TurnPhase = "WAIT_DRAW"
	r.Updated = time.Now().Unix()

	if len(r.DrawPile) == 0 {
		// ✅ el bitti (son discard sonrası)
		r.endHandLocked()
		r.Updated = time.Now().Unix()
		return nil
	}

	r.resetTurnTimerLocked()
	return nil

}

/* =========================
   HTTP + WS
   ========================= */

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

type HelloPayload struct{ UserID string `json:"userId"` }
type RoomCreatePayload struct {
    UserID string `json:"userId"`
    Config *RoomConfig `json:"config,omitempty"`
}

type RoomJoinPayload struct {
	UserID string `json:"userId"`
	RoomID string `json:"roomId"`
}
type DiceStopPayload struct {
	UserID string `json:"userId"`
	RoomID string `json:"roomId"`
}
type DrawPayload struct {
	UserID string `json:"userId"`
	RoomID string `json:"roomId"`
}
type DiscardPayload struct {
	UserID string `json:"userId"`
	RoomID string `json:"roomId"`
	TileID string `json:"tileId"`
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS upgrade error:", err)
		return
	}
	c := &Conn{ws: ws, send: make(chan []byte, 64)}
	go writePump(c)
	readPump(c)
}

func normalizeConfig(in *RoomConfig) (RoomConfig, error) {
    // default
    cfg := RoomConfig{
        GameMode:    GameModeClassic,
        PenaltyMode: PenaltyOn,
        HandCount:   1,
    }
    if in == nil {
        return cfg, nil
    }

    if in.GameMode != "" {
        if in.GameMode != GameModeClassic && in.GameMode != GameModeKatlamali {
            return cfg, errors.New("invalid gameMode")
        }
        cfg.GameMode = in.GameMode
    }

    if in.PenaltyMode != "" {
        if in.PenaltyMode != PenaltyOn && in.PenaltyMode != PenaltyOff {
            return cfg, errors.New("invalid penaltyMode")
        }
        cfg.PenaltyMode = in.PenaltyMode
    }

    if in.HandCount != 0 {
        if in.HandCount < 1 || in.HandCount > 11 {
            return cfg, errors.New("handCount must be 1..11")
        }
        cfg.HandCount = in.HandCount
    }

    return cfg, nil
}


func readPump(c *Conn) {
    defer func() {

        // ✅ 1) Lobby listesinden çıkar (eğer lobbyConns eklediysen)
        rooms.mu.Lock()
        delete(rooms.lobbyConns, c)
        rooms.mu.Unlock()

        // ✅ 2) Odadaysa odadan ayır
        if c.roomID != "" && c.userID != "" {
            if room, ok := rooms.GetRoom(c.roomID); ok {
                room.detachConn(c.userID)
                room.broadcastSnapshot()
            }
        }

        // ✅ 3) WS kapat
        _ = c.ws.Close()

        // ✅ 4) Lobby'ye yeni listeyi yayınla
        rooms.BroadcastRoomsList()
    }()

	_ = c.ws.SetReadDeadline(time.Now().Add(120 * time.Second))
	c.ws.SetPongHandler(func(string) error {
		_ = c.ws.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			log.Println("WS read error:", err)
			return
		}

		var in InMsg
		if err := json.Unmarshal(data, &in); err != nil {
			sendErr(c, "", "BAD_JSON", "invalid json")
			continue
		}

		switch in.T {
		case "PING":
			send(c, OutMsg{T: "PONG", ReqID: in.ReqID})

		case "HELLO":
			var p HelloPayload
			_ = json.Unmarshal(in.P, &p)
			if p.UserID == "" {
				sendErr(c, in.ReqID, "MISSING_USER", "userId required")
				continue
			}

			c.userID = p.UserID

			// ✅ lobby kaydı (bu conn lobby’de masaları görebilsin)
			rooms.mu.Lock()
			rooms.lobbyConns[c] = true
			rooms.mu.Unlock()

			// HELLO_OK
			send(c, OutMsg{
				T:     "HELLO_OK",
				ReqID: in.ReqID,
				P:     map[string]any{"userId": c.userID},
			})

			// ✅ bağlanır bağlanmaz mevcut masaları yolla (lobby list)
			rooms.BroadcastRoomsList()


		case "ROOM_CREATE":
			var p RoomCreatePayload
			_ = json.Unmarshal(in.P, &p)

			// userId: payload'tan gelmezse HELLO ile geleni kullan
			uid := p.UserID
			if uid == "" {
				uid = c.userID
			}
			if uid == "" {
				sendErr(c, in.ReqID, "MISSING_USER", "send HELLO first or include userId in payload")
				continue
			}
			c.userID = uid

			// config doğrula + default bas
			cfg, err := normalizeConfig(p.Config)
			if err != nil {
				sendErr(c, in.ReqID, "BAD_CONFIG", err.Error())
				continue
			}

			// Odayı config ile oluştur
			room, err := rooms.CreateRoom(uid, cfg)
			if err != nil {
				sendErr(c, in.ReqID, "CREATE_FAILED", err.Error())
				continue
			}

			// conn'i odaya bağla
			c.roomID = room.ID
			room.attachConn(uid, c)

			// cevap + snapshot
			send(c, OutMsg{
				T:     "ROOM_CREATED",
				ReqID: in.ReqID,
				P:     map[string]any{"roomId": room.ID},
			})
			room.broadcastSnapshot()
			rooms.BroadcastRoomsList()



		case "ROOM_JOIN":
			var p RoomJoinPayload
			_ = json.Unmarshal(in.P, &p)

			uid := p.UserID
			if uid == "" { uid = c.userID }
			if uid == "" {
				sendErr(c, in.ReqID, "MISSING_USER", "send HELLO first or include userId")
				continue
			}
			if p.RoomID == "" {
				sendErr(c, in.ReqID, "MISSING_ROOM", "roomId required")
				continue
			}
			c.userID = uid

			room, ok := rooms.GetRoom(p.RoomID)
			if !ok {
				sendErr(c, in.ReqID, "ROOM_NOT_FOUND", "room not found")
				continue
			}

			// ✅ 1 userId = 1 aktif masa kuralı
			// - başka bir odadaysa JOIN reddedilir
			// - aynı odaya (reconnect) izin vardır
			if err := rooms.ReserveUserRoom(uid, room.ID); err != nil {
				sendErr(c, in.ReqID, "ALREADY_IN_ROOM", err.Error())
				continue
			}

			_, err := room.join(uid)
			if err != nil {
				// join fail olursa rezervasyonu geri al
				rooms.ReleaseUserRoom(uid, room.ID)
				sendErr(c, in.ReqID, "JOIN_FAILED", err.Error())
				continue
			}

			c.roomID = room.ID
			room.attachConn(uid, c)

			send(c, OutMsg{T: "ROOM_JOINED", ReqID: in.ReqID, P: map[string]any{"roomId": room.ID}})
			room.broadcastSnapshot()
			rooms.BroadcastRoomsList()



		case "DICE_STOP":
			var p DiceStopPayload
			_ = json.Unmarshal(in.P, &p)

			uid := p.UserID
			if uid == "" { uid = c.userID }
			if uid == "" {
				sendErr(c, in.ReqID, "MISSING_USER", "userId required")
				continue
			}
			roomID := p.RoomID
			if roomID == "" { roomID = c.roomID }
			if roomID == "" {
				sendErr(c, in.ReqID, "MISSING_ROOM", "roomId required")
				continue
			}

			room, ok := rooms.GetRoom(roomID)
			if !ok {
				sendErr(c, in.ReqID, "ROOM_NOT_FOUND", "room not found")
				continue
			}
			if err := room.diceStop(uid); err != nil {
				sendErr(c, in.ReqID, "DICE_STOP_REJECTED", err.Error())
				continue
			}
			room.broadcastSnapshot()

		case "DRAW":
			var p DrawPayload
			_ = json.Unmarshal(in.P, &p)
			uid := p.UserID
			if uid == "" { uid = c.userID }
			if uid == "" {
				sendErr(c, in.ReqID, "MISSING_USER", "userId required")
				continue
			}
			roomID := p.RoomID
			if roomID == "" { roomID = c.roomID }
			if roomID == "" {
				sendErr(c, in.ReqID, "MISSING_ROOM", "roomId required")
				continue
			}
			room, ok := rooms.GetRoom(roomID)
			if !ok {
				sendErr(c, in.ReqID, "ROOM_NOT_FOUND", "room not found")
				continue
			}
			if err := room.draw(uid); err != nil {
				sendErr(c, in.ReqID, "DRAW_REJECTED", err.Error())
				continue
			}
			room.broadcastSnapshot()

		case "DISCARD":
			var p DiscardPayload
			_ = json.Unmarshal(in.P, &p)
			uid := p.UserID
			if uid == "" { uid = c.userID }
			if uid == "" {
				sendErr(c, in.ReqID, "MISSING_USER", "userId required")
				continue
			}
			roomID := p.RoomID
			if roomID == "" { roomID = c.roomID }
			if roomID == "" {
				sendErr(c, in.ReqID, "MISSING_ROOM", "roomId required")
				continue
			}
			room, ok := rooms.GetRoom(roomID)
			if !ok {
				sendErr(c, in.ReqID, "ROOM_NOT_FOUND", "room not found")
				continue
			}
			if err := room.discard(uid, p.TileID); err != nil {
				sendErr(c, in.ReqID, "DISCARD_REJECTED", err.Error())
				continue
			}
			room.broadcastSnapshot()



		case "ROOMS_LIST_REQUEST":
			rooms.BroadcastRoomsList()

		case "MELD_SUGGEST":
			var p struct {
				RoomID string `json:"roomId"`
				UserID string `json:"userId"`
				Mode   string `json:"mode"` // "RUN" | "PAIR" | "" (AUTO)
			}
			_ = json.Unmarshal(in.P, &p)

			if p.RoomID == "" || p.UserID == "" {
				sendErr(c, in.ReqID, "BAD_REQUEST", "roomId and userId required")
				continue
			}

			r, ok := rooms.GetRoom(p.RoomID)
			if !ok {
				sendErr(c, in.ReqID, "ROOM_NOT_FOUND", "room not found")
				continue
			}

			if c.userID != p.UserID {
				sendErr(c, in.ReqID, "FORBIDDEN", "user mismatch")
				continue
			}

			seat := r.seatOf(p.UserID)
			if seat == 0 {
				sendErr(c, in.ReqID, "NOT_IN_ROOM", "user not seated")
				continue
			}

			// --- eldeki taşları al
			r.mu.RLock()
			hand := append([]string(nil), r.Hands[seat]...)
			indicator := r.Indicator       // örn "R07-1"
			realOkeyBase := r.OkeyTileID   // örn "R08"
			r.mu.RUnlock()

			hh := handHash(hand)
			mode := strings.ToUpper(strings.TrimSpace(p.Mode))
			if mode == "" {
			mode = "AUTO"
			}

			cacheKey := p.UserID + ":" + hh + ":" + mode


			// --- cache kontrol
			r.mu.RLock()
			cached, ok := r.SolverCache[cacheKey]
			r.mu.RUnlock()

			if ok {
				send(c, OutMsg{
					T:     "MELD_SUGGESTED",
					ReqID: in.ReqID,
					P: map[string]any{
						"roomId":   p.RoomID,
						"userId":   p.UserID,
						"handHash": hh,
						"cached":   true,
						"result":   cached,
					},
				})
				continue
			}

			// --- solver çağrısı (SADECE DİZİM)
			solveMode := SolveAuto
			switch mode {
			case "RUN":
			solveMode = SolveRun
			case "PAIR":
			solveMode = SolvePair
			case "AUTO":
			solveMode = SolveAuto
			default:
			solveMode = SolveAuto
			}

			res := SuggestMelds(
			hand,
			indicator,
			realOkeyBase,
			solveMode,
			30*time.Millisecond,
			)


			// --- cache yaz
			r.mu.Lock()
			r.SolverCache[cacheKey] = res
			r.mu.Unlock()

			send(c, OutMsg{
				T:     "MELD_SUGGESTED",
				ReqID: in.ReqID,
				P: map[string]any{
					"roomId":   p.RoomID,
					"userId":   p.UserID,
					"handHash": hh,
					"cached":   false,
					"result":   res,
				},
			})







		default:
			sendErr(c, in.ReqID, "UNKNOWN_TYPE", "unknown message type: "+in.T)
		}
	}
}

func writePump(c *Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.ws.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok { return }
			_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil { return }
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil { return }
		}
	}
}

func send(c *Conn, out OutMsg) {
	b, _ := json.Marshal(out)
	select { case c.send <- b: default: }
}
func sendErr(c *Conn, reqID, code, msg string) {
	send(c, OutMsg{T: "ERROR", ReqID: reqID, P: ErrPayload{Code: code, Msg: msg}})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ws", wsHandler)

	log.Println("API listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
