package search

import (
	"context"

	"sloppy-disk/internal/albumcache"
	"sloppy-disk/internal/models"
)

func FuzzySearch(ctx context.Context, query string) ([]models.Album, error) {
	cache := albumcache.GetCache()
	results, _ := cache.SearchAlbums(query, 0, 50)
	return results, nil
}

func EnhancedFuzzySearch(ctx context.Context, query string, page, limit int) ([]models.Album, error) {
	cache := albumcache.GetCache()
	offset := (page - 1) * limit
	results, _ := cache.SearchAlbums(query, offset, limit)
	return results, nil
}
