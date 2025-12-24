package service

import (
	"fmt"
	"html"
	"log"
	"os"
	"strings"
	"time"

	"anoa.com/telkomalumiforum/internal/model"
	"github.com/meilisearch/meilisearch-go"
	"github.com/microcosm-cc/bluemonday"
)

type MeiliSearchService interface {
	IndexThread(thread *model.Thread) error
	IndexPost(post *model.Post) error
	DeleteThread(id string) error
	DeletePost(id string) error
	GenerateSearchToken(userRole string) (string, error)
}

type meiliSearchService struct {
	client        meilisearch.ServiceManager
	masterKey     string
	signingKeyUID string
	signingKey    string
	sanitizer     *bluemonday.Policy
}

func NewMeiliSearchService(client meilisearch.ServiceManager) MeiliSearchService {
	masterKey := os.Getenv("MEILI_MASTER_KEY")
	if masterKey == "" {
		log.Println("WARNING: MEILI_MASTER_KEY is not set.")
	}

	s := &meiliSearchService{
		client:    client,
		masterKey: masterKey,
		sanitizer: bluemonday.StrictPolicy(),
	}
	s.initIndexes()
	s.initSigningKey()
	return s
}

func (s *meiliSearchService) initSigningKey() {
	// 1. List keys
	resp, err := s.client.GetKeys(&meilisearch.KeysQuery{
		Limit: 20,
	})
	if err != nil {
		log.Printf("Failed to get meilisearch keys: %v", err)
		return
	}

	// 2. Find existing key for signing
	for _, key := range resp.Results {
		if key.Name == "TenantTokenSigner" {
			s.signingKeyUID = key.UID
			s.signingKey = key.Key
			log.Println("Found existing Meilisearch signing key")
			return
		}
	}

	// 3. Create new key if not found
	// Expiry: nil (forever) or long time
	expiry := time.Now().AddDate(100, 0, 0)

	key, err := s.client.CreateKey(&meilisearch.Key{
		Description: "Key to sign tenant tokens",
		Name:        "TenantTokenSigner",
		Actions:     []string{"search"},
		Indexes:     []string{"threads", "posts"},
		ExpiresAt:   expiry,
	})
	if err != nil {
		log.Printf("Failed to create signing key: %v", err)
		return
	}

	s.signingKeyUID = key.UID
	s.signingKey = key.Key
	log.Println("Created new Meilisearch signing key")
}

func (s *meiliSearchService) initIndexes() {
	// Threads Index
	filterableAttrs := []string{"allowed_roles", "category_id"}
	filterableInterface := make([]any, len(filterableAttrs))
	for i, v := range filterableAttrs {
		filterableInterface[i] = v
	}
	_, err := s.client.Index("threads").UpdateFilterableAttributes(&filterableInterface)
	if err != nil {
		log.Printf("Failed to update threads filterable attributes: %v", err)
	}

	sortableAttrs := []string{"created_at", "views"}
	_, err = s.client.Index("threads").UpdateSortableAttributes(&sortableAttrs)
	if err != nil {
		log.Printf("Failed to update threads sortable attributes: %v", err)
	}

	// Posts Index
	postFilterable := []string{"allowed_roles", "thread_id"}
	postFilterableInterface := make([]any, len(postFilterable))
	for i, v := range postFilterable {
		postFilterableInterface[i] = v
	}
	_, err = s.client.Index("posts").UpdateFilterableAttributes(&postFilterableInterface)
	if err != nil {
		log.Printf("Failed to update posts filterable attributes: %v", err)
	}

	postSortable := []string{"created_at"}
	_, err = s.client.Index("posts").UpdateSortableAttributes(&postSortable)
	if err != nil {
		log.Printf("Failed to update posts sortable attributes: %v", err)
	}

	log.Println("Meilisearch indexes initialized")
}

// Structs for Meilisearch Indexing
type meiliThreadDoc struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	Content      string              `json:"content"`
	Slug         string              `json:"slug"`
	Audience     string              `json:"audience"`
	AllowedRoles []string            `json:"allowed_roles"`
	Views        int                 `json:"views"`
	CreatedAt    int64               `json:"created_at"`
	CategoryID   string              `json:"category_id"`
	User         meiliUserSubset     `json:"user"`
	Category     meiliCategorySubset `json:"category"`
}

type meiliPostDoc struct {
	ID           string          `json:"id"`
	Content      string          `json:"content"`
	ThreadID     string          `json:"thread_id"`
	ThreadSlug   string          `json:"thread_slug"`
	ThreadTitle  string          `json:"thread_title"`
	AllowedRoles []string        `json:"allowed_roles"`
	CreatedAt    int64           `json:"created_at"`
	User         meiliUserSubset `json:"user"`
}

type meiliUserSubset struct {
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

type meiliCategorySubset struct {
	Name string `json:"name"`
}

func (s *meiliSearchService) cleanContentForIndex(content string) string {
	// 1. Replace block tags with spaces to prevent text merging
	content = strings.ReplaceAll(content, "</p>", " ")
	content = strings.ReplaceAll(content, "<br>", " ")
	content = strings.ReplaceAll(content, "</div>", " ")

	// 2. Sanitize
	sanitized := s.sanitizer.Sanitize(content)

	// 3. Unescape entities
	cleanText := html.UnescapeString(sanitized)

	// 4. Normalize whitespace
	cleanText = strings.Join(strings.Fields(cleanText), " ")

	return cleanText
}

func (s *meiliSearchService) IndexThread(thread *model.Thread) error {
	allowedRoles := []string{}
	if thread.Audience == "semua" {
		allowedRoles = append(allowedRoles, "public")
	} else {
		allowedRoles = append(allowedRoles, thread.Audience)
	}

	doc := meiliThreadDoc{
		ID:           thread.ID.String(),
		Title:        thread.Title,
		Content:      s.cleanContentForIndex(thread.Content),
		Slug:         thread.Slug,
		Audience:     thread.Audience,
		AllowedRoles: allowedRoles,
		Views:        thread.Views,
		CreatedAt:    thread.CreatedAt.Unix(),
		CategoryID:   thread.CategoryID.String(), // Dereference if needed, but assuming model has *UUID handled or assuming it's UUID value. Double check model.
		User: meiliUserSubset{
			Username:  thread.User.Username,
			AvatarURL: getStringOrEmpty(thread.User.AvatarURL),
		},
		Category: meiliCategorySubset{
			Name: thread.Category.Name,
		},
	}

	// Double check CategoryID type in model.
	// In model: CategoryID *uuid.UUID.
	if thread.CategoryID != nil {
		doc.CategoryID = thread.CategoryID.String()
	}

	// Debug log
	log.Printf("Indexing thread document: %+v", doc)

	task, err := s.client.Index("threads").AddDocuments([]meiliThreadDoc{doc}, strPtr("id"))
	if err != nil {
		return err
	}
	log.Printf("Indexed thread %s, task id: %d", thread.ID, task.TaskUID)
	return nil
}

func (s *meiliSearchService) IndexPost(post *model.Post) error {

	if post.Thread.ID.String() == "00000000-0000-0000-0000-000000000000" {
		log.Println("Warning: IndexPost called with missing Thread relation. Audience might be wrong.")
		return fmt.Errorf("post thread not loaded")
	}

	allowedRoles := []string{}
	if post.Thread.Audience == "semua" {
		allowedRoles = append(allowedRoles, "public")
	} else {
		allowedRoles = append(allowedRoles, post.Thread.Audience)
	}

	log.Println("Indexing post document: ", post)

	doc := meiliPostDoc{
		ID:           post.ID.String(),
		Content:      s.cleanContentForIndex(post.Content),
		ThreadID:     post.ThreadID.String(),
		ThreadSlug:   post.Thread.Slug,
		ThreadTitle:  post.Thread.Title,
		AllowedRoles: allowedRoles,
		CreatedAt:    post.CreatedAt.Unix(),
		User: meiliUserSubset{
			Username:  post.User.Username,
			AvatarURL: getStringOrEmpty(post.User.AvatarURL),
		},
	}

	log.Println("Indexed post document after sanitize: ", doc)

	task, err := s.client.Index("posts").AddDocuments([]meiliPostDoc{doc}, strPtr("id"))
	if err != nil {
		return err
	}
	log.Printf("Indexed post %s, task id: %d", post.ID, task.TaskUID)
	return nil
}

func getStringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *meiliSearchService) DeleteThread(id string) error {
	_, err := s.client.Index("threads").DeleteDocument(id)
	return err
}

func (s *meiliSearchService) DeletePost(id string) error {
	_, err := s.client.Index("posts").DeleteDocument(id)
	return err
}

func (s *meiliSearchService) GenerateSearchToken(userRole string) (string, error) {
	if s.signingKeyUID == "" || s.signingKey == "" {
		return "", fmt.Errorf("signing key not initialized")
	}

	// Rules based on role
	var filterRules string

	switch userRole {
	case "admin":
		filterRules = "" // No filter
	case "guru":
		filterRules = "allowed_roles IN ['guru', 'public']"
	default:
		// Siswa and others
		filterRules = "allowed_roles IN ['siswa', 'public']"
	}

	searchRules := map[string]any{
		"threads": map[string]any{},
		"posts":   map[string]any{},
	}

	if filterRules != "" {
		searchRules["threads"] = map[string]any{
			"filter": filterRules,
		}
		searchRules["posts"] = map[string]any{
			"filter": filterRules,
		}
	} else {
		// For admin (empty filterRules), we give full access.
		searchRules["threads"] = map[string]any{"filter": nil}
		searchRules["posts"] = map[string]any{"filter": nil}
	}

	// Use the dedicated signing Key
	token, err := s.client.GenerateTenantToken(s.signingKeyUID, searchRules, &meilisearch.TenantTokenOptions{
		APIKey:    s.signingKey,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	if err != nil {
		return "", err
	}

	return token, nil
}

func strPtr(s string) *string {
	return &s
}
