package routes

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const pluginAssetCursorMaxEntries = 512
const pluginAssetCursorMaxBytes = 8 << 20
const pluginAssetCursorMaxEntriesPerUser = 8
const pluginAssetCursorMaxStateBytes = 4 << 10
const pluginAssetCursorMaxBatches = 50
const pluginAssetCursorTTL = 10 * time.Minute

type pluginAssetCursorBinding struct {
	UserID            string
	ActorID           string
	PluginID          string
	InstanceID        string
	QueryFingerprint  string
	ConfigFingerprint string
}

type pluginAssetCursorEntry struct {
	Binding   pluginAssetCursorBinding
	State     map[string]any
	StateHash string
	Seen      map[string]bool
	Batches   int
	Size      int
	ExpiresAt time.Time
	LastUsed  time.Time
}

type pluginAssetCursorStore struct {
	sync.Mutex
	items      map[string]pluginAssetCursorEntry
	totalBytes int
}

var assetPluginCursors = pluginAssetCursorStore{items: map[string]pluginAssetCursorEntry{}}

func newPluginAssetCursorBinding(userID, actorID, pluginID, instanceID string, request pluginAssetLibraryActionInput, auth, config map[string]any) (pluginAssetCursorBinding, error) {
	queryFingerprint, err := pluginAssetCanonicalFingerprint(request)
	if err != nil {
		return pluginAssetCursorBinding{}, err
	}
	configFingerprint, err := pluginAssetCanonicalFingerprint(struct {
		Auth   map[string]any `json:"auth,omitempty"`
		Config map[string]any `json:"config,omitempty"`
	}{Auth: auth, Config: config})
	if err != nil {
		return pluginAssetCursorBinding{}, err
	}
	return pluginAssetCursorBinding{
		UserID:            strings.TrimSpace(userID),
		ActorID:           strings.TrimSpace(actorID),
		PluginID:          strings.TrimSpace(pluginID),
		InstanceID:        strings.TrimSpace(instanceID),
		QueryFingerprint:  queryFingerprint,
		ConfigFingerprint: configFingerprint,
	}, nil
}

func pluginAssetCanonicalFingerprint(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func pluginAssetStateHash(state map[string]any) (string, []byte, error) {
	if len(state) == 0 {
		return "", nil, nil
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		return "", nil, fmt.Errorf("asset plugin returned invalid continuation state: %w", err)
	}
	if len(encoded) > pluginAssetCursorMaxStateBytes {
		return "", nil, fmt.Errorf("asset plugin continuation state exceeds %d bytes", pluginAssetCursorMaxStateBytes)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), encoded, nil
}

func validatePluginAssetContinuation(inputState, outputState map[string]any, hasMore bool, seen map[string]bool) (string, []byte, error) {
	if !hasMore {
		return "", nil, nil
	}
	outputHash, encoded, err := pluginAssetStateHash(outputState)
	if err != nil {
		return "", nil, err
	}
	if outputHash == "" {
		return "", nil, fmt.Errorf("asset plugin returned hasMore=true without continuation state")
	}
	inputHash, _, err := pluginAssetStateHash(inputState)
	if err != nil {
		return "", nil, err
	}
	if inputHash != "" && outputHash == inputHash {
		return "", nil, fmt.Errorf("asset plugin returned non-progressing continuation state")
	}
	if seen[outputHash] {
		return "", nil, fmt.Errorf("asset plugin returned cyclic continuation state")
	}
	return outputHash, encoded, nil
}

func (s *pluginAssetCursorStore) resume(id string, binding pluginAssetCursorBinding) (pluginAssetCursorEntry, bool, error) {
	s.Lock()
	defer s.Unlock()
	now := time.Now()
	s.pruneExpiredLocked(now)
	entry, ok := s.items[id]
	if !ok {
		return pluginAssetCursorEntry{}, true, nil
	}
	if entry.Binding != binding {
		if entry.Binding.UserID == binding.UserID && entry.Binding.ActorID == binding.ActorID {
			s.deleteLocked(id)
		}
		return pluginAssetCursorEntry{}, true, nil
	}
	if entry.Batches >= pluginAssetCursorMaxBatches {
		return pluginAssetCursorEntry{}, false, fmt.Errorf("asset plugin cursor exceeded %d batches", pluginAssetCursorMaxBatches)
	}
	entry.LastUsed = now
	entry.ExpiresAt = now.Add(pluginAssetCursorTTL)
	s.items[id] = entry
	clonedState, err := clonePluginAssetState(entry.State)
	if err != nil {
		return pluginAssetCursorEntry{}, false, err
	}
	entry.State = clonedState
	entry.Seen = clonePluginAssetSeen(entry.Seen)
	return entry, false, nil
}

func (s *pluginAssetCursorStore) create(binding pluginAssetCursorBinding, state map[string]any, stateHash string) (string, error) {
	clonedState, err := clonePluginAssetState(state)
	if err != nil {
		return "", err
	}
	now := time.Now()
	entry := pluginAssetCursorEntry{
		Binding:   binding,
		State:     clonedState,
		StateHash: stateHash,
		Seen:      map[string]bool{stateHash: true},
		Batches:   1,
		ExpiresAt: now.Add(pluginAssetCursorTTL),
		LastUsed:  now,
	}
	s.Lock()
	defer s.Unlock()
	s.pruneExpiredLocked(now)
	id := ""
	for id == "" {
		idBytes := make([]byte, 16)
		if _, err := rand.Read(idBytes); err != nil {
			return "", err
		}
		candidate := hex.EncodeToString(idBytes)
		if _, exists := s.items[candidate]; !exists {
			id = candidate
		}
	}
	entry.Size = pluginAssetCursorEntrySize(id, entry)
	for s.userEntryCountLocked(binding.UserID) >= pluginAssetCursorMaxEntriesPerUser {
		if !s.evictOldestLocked(binding.UserID) {
			break
		}
	}
	s.items[id] = entry
	s.totalBytes += entry.Size
	for len(s.items) > pluginAssetCursorMaxEntries || s.totalBytes > pluginAssetCursorMaxBytes {
		if !s.evictOldestLocked("") {
			break
		}
	}
	return id, nil
}

func (s *pluginAssetCursorStore) advance(id string, binding pluginAssetCursorBinding, expectedStateHash string, state map[string]any, stateHash string, seen map[string]bool, hasMore bool) (bool, error) {
	s.Lock()
	defer s.Unlock()
	entry, ok := s.items[id]
	if !ok || entry.Binding != binding || entry.StateHash != expectedStateHash {
		return true, nil
	}
	if !hasMore {
		s.deleteLocked(id)
		return false, nil
	}
	if entry.Batches+1 > pluginAssetCursorMaxBatches {
		s.deleteLocked(id)
		return false, fmt.Errorf("asset plugin cursor exceeded %d batches", pluginAssetCursorMaxBatches)
	}
	clonedState, err := clonePluginAssetState(state)
	if err != nil {
		return false, err
	}
	entry.State = clonedState
	entry.StateHash = stateHash
	entry.Seen = clonePluginAssetSeen(seen)
	entry.Batches++
	entry.LastUsed = time.Now()
	entry.ExpiresAt = entry.LastUsed.Add(pluginAssetCursorTTL)
	previousSize := entry.Size
	entry.Size = pluginAssetCursorEntrySize(id, entry)
	s.totalBytes += entry.Size - previousSize
	s.items[id] = entry
	for s.totalBytes > pluginAssetCursorMaxBytes {
		if !s.evictOldestLocked("") {
			break
		}
	}
	return false, nil
}

func (s *pluginAssetCursorStore) deleteLocked(id string) {
	entry, ok := s.items[id]
	if !ok {
		return
	}
	delete(s.items, id)
	s.totalBytes -= entry.Size
}

func (s *pluginAssetCursorStore) pruneExpiredLocked(now time.Time) {
	for id, entry := range s.items {
		if !entry.ExpiresAt.After(now) {
			s.deleteLocked(id)
		}
	}
}

func (s *pluginAssetCursorStore) userEntryCountLocked(userID string) int {
	count := 0
	for _, entry := range s.items {
		if entry.Binding.UserID == userID {
			count++
		}
	}
	return count
}

func (s *pluginAssetCursorStore) evictOldestLocked(userID string) bool {
	oldestID := ""
	var oldest time.Time
	for id, entry := range s.items {
		if userID != "" && entry.Binding.UserID != userID {
			continue
		}
		if oldestID == "" || entry.LastUsed.Before(oldest) {
			oldestID = id
			oldest = entry.LastUsed
		}
	}
	if oldestID == "" {
		return false
	}
	s.deleteLocked(oldestID)
	return true
}

func clonePluginAssetState(state map[string]any) (map[string]any, error) {
	if len(state) == 0 {
		return nil, nil
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	var cloned map[string]any
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return nil, err
	}
	return cloned, nil
}

func clonePluginAssetSeen(seen map[string]bool) map[string]bool {
	cloned := make(map[string]bool, len(seen))
	for value := range seen {
		cloned[value] = true
	}
	return cloned
}

func pluginAssetCursorEntrySize(id string, entry pluginAssetCursorEntry) int {
	size := len(id)
	size += len(entry.Binding.UserID) + len(entry.Binding.ActorID) + len(entry.Binding.PluginID) + len(entry.Binding.InstanceID)
	size += len(entry.Binding.QueryFingerprint) + len(entry.Binding.ConfigFingerprint) + len(entry.StateHash)
	state, _ := json.Marshal(entry.State)
	size += len(state)
	for hash := range entry.Seen {
		size += len(hash)
	}
	return size
}
