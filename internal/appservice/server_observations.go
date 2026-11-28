package appservice

import (
	"math"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

const (
	maxObservedPlaces          = 16
	maxObservedServersPerPlace = 2000
	maxServerListings          = 30
	serverObservationKeep      = time.Hour
	// Listings closer together than this, such as two pages that list the
	// same server, count once.
	minServerListingInterval = 10 * time.Second
)

type serverListing struct {
	atMs    int64
	playing int64
	fps     float64
	pingMs  *int64
}

type observedPlace struct {
	lastListedMs int64
	serverByID   map[string][]serverListing
}

// serverObservations remembers, in memory only, each time that the Roblox
// server list showed a server, so that a server can show how it changed.
type serverObservations struct {
	mutex     sync.Mutex
	placeByID map[int64]*observedPlace
}

func newServerObservations() *serverObservations {
	return &serverObservations{placeByID: make(map[int64]*observedPlace)}
}

// record adds the listed servers and sets the observation of each.
func (observations *serverObservations) record(placeID int64, servers []games.Server, now time.Time) {
	observations.mutex.Lock()
	defer observations.mutex.Unlock()
	nowMs := now.UnixMilli()
	place := observations.place(placeID, nowMs)
	for index := range servers {
		server := &servers[index]
		listings := place.serverByID[server.ServerID]
		listing := serverListing{atMs: nowMs, playing: server.Playing, fps: server.FPS, pingMs: server.PingMs}
		if count := len(listings); count > 0 && nowMs-listings[count-1].atMs < minServerListingInterval.Milliseconds() {
			listings[count-1] = listing
		} else {
			if count == maxServerListings {
				listings = listings[1:]
			}
			listings = append(listings, listing)
		}
		if _, isListed := place.serverByID[server.ServerID]; isListed || len(place.serverByID) < maxObservedServersPerPlace {
			place.serverByID[server.ServerID] = listings
		}
		server.Observation = summarizeListings(listings)
	}
}

// place returns the observed place, and forgets old servers and the place
// listed longest ago when too many places are kept.
func (observations *serverObservations) place(placeID, nowMs int64) *observedPlace {
	for id, kept := range observations.placeByID {
		if nowMs-kept.lastListedMs > serverObservationKeep.Milliseconds() {
			delete(observations.placeByID, id)
		}
	}
	place, found := observations.placeByID[placeID]
	if !found {
		if len(observations.placeByID) >= maxObservedPlaces {
			oldestID, oldestMs := int64(0), int64(math.MaxInt64)
			for id, kept := range observations.placeByID {
				if kept.lastListedMs < oldestMs {
					oldestID, oldestMs = id, kept.lastListedMs
				}
			}
			delete(observations.placeByID, oldestID)
		}
		place = &observedPlace{serverByID: make(map[string][]serverListing)}
		observations.placeByID[placeID] = place
	}
	place.lastListedMs = nowMs
	for serverID, listings := range place.serverByID {
		if nowMs-listings[len(listings)-1].atMs > serverObservationKeep.Milliseconds() {
			delete(place.serverByID, serverID)
		}
	}
	return place
}

func summarizeListings(listings []serverListing) *games.ServerObservation {
	if len(listings) == 0 {
		return nil
	}
	first, last := listings[0], listings[len(listings)-1]
	observation := &games.ServerObservation{
		FirstSeenMs: first.atMs, LastSeenMs: last.atMs, ListingCount: len(listings),
		PlayingChange: last.playing - first.playing, PlayingHistory: make([]int64, len(listings)),
	}
	framesPerSecond := make([]float64, len(listings))
	pings := make([]float64, 0, len(listings))
	for index, listing := range listings {
		observation.PlayingHistory[index] = listing.playing
		framesPerSecond[index] = listing.fps
		if listing.pingMs != nil {
			pings = append(pings, float64(*listing.pingMs))
		}
	}
	observation.FPS = observedRange(framesPerSecond)
	if len(pings) > 0 {
		pingRange := observedRange(pings)
		observation.PingMs = &pingRange
	}
	return observation
}

func observedRange(samples []float64) games.ObservedRange {
	summary := games.ObservedRange{Minimum: samples[0], Maximum: samples[0]}
	total := 0.0
	for _, sample := range samples {
		summary.Minimum, summary.Maximum = min(summary.Minimum, sample), max(summary.Maximum, sample)
		total += sample
	}
	summary.Average = total / float64(len(samples))
	variance := 0.0
	for _, sample := range samples {
		variance += (sample - summary.Average) * (sample - summary.Average)
	}
	summary.StandardDeviation = math.Sqrt(variance / float64(len(samples)))
	return summary
}
