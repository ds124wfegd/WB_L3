package database

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ds124wfegd/WB_L3/3/internal/entity"
	"github.com/redis/go-redis/v9"
)

type CommentRepository struct {
	client *redis.Client
	ctx    context.Context
}

func NewCommentRepository(redisClient *redis.Client) (*CommentRepository, error) {

	ctx := context.Background()

	// Проверка подключения
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &CommentRepository{
		client: redisClient,
		ctx:    ctx,
	}, nil
}

func (r *CommentRepository) Create(comment entity.Comment) error {
	// Сохраняем комментарий
	commentKey := fmt.Sprintf("comment:%s", comment.ID)
	if err := r.client.Set(r.ctx, commentKey, &comment, 0).Err(); err != nil {
		return err
	}

	// Добавляем в индекс по родителю
	if comment.ParentID == "" {
		// Корневой комментарий
		if err := r.client.SAdd(r.ctx, "comments:root", comment.ID).Err(); err != nil {
			return err
		}
	} else {
		// Дочерний комментарий
		parentKey := fmt.Sprintf("comment:%s:children", comment.ParentID)
		if err := r.client.SAdd(r.ctx, parentKey, comment.ID).Err(); err != nil {
			return err
		}
	}

	// Добавляем в индекс для поиска
	searchKey := "comments:all"
	if err := r.client.SAdd(r.ctx, searchKey, comment.ID).Err(); err != nil {
		return err
	}

	// Индексируем для поиска по тексту и автору
	if err := r.indexCommentForSearch(&comment); err != nil {
		return err
	}

	return nil
}

func (r *CommentRepository) GetByID(id string) (*entity.Comment, bool) {
	commentKey := fmt.Sprintf("comment:%s", id)
	data, err := r.client.Get(r.ctx, commentKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		return nil, false
	}

	var comment entity.Comment
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, false
	}

	return &comment, true
}

func (r *CommentRepository) GetChildren(parentID string, page, pageSize int, sortBy string) ([]entity.Comment, int) {
	var children []entity.Comment
	var childIDs []string

	if parentID == "" {
		// Корневые комментарии
		ids, err := r.client.SMembers(r.ctx, "comments:root").Result()
		if err != nil {
			return children, 0
		}
		childIDs = ids
	} else {
		// Дочерние комментарии
		parentKey := fmt.Sprintf("comment:%s:children", parentID)
		ids, err := r.client.SMembers(r.ctx, parentKey).Result()
		if err != nil {
			return children, 0
		}
		childIDs = ids
	}

	// Получаем комментарии по ID
	for _, id := range childIDs {
		if comment, exists := r.GetByID(id); exists {
			children = append(children, *comment)
		}
	}

	// Сортировка
	switch sortBy {
	case "created_at_desc":
		sort.Slice(children, func(i, j int) bool {
			return children[i].CreatedAt.After(children[j].CreatedAt)
		})
	case "created_at_asc":
		sort.Slice(children, func(i, j int) bool {
			return children[i].CreatedAt.Before(children[j].CreatedAt)
		})
	case "author":
		sort.Slice(children, func(i, j int) bool {
			return strings.ToLower(children[i].Author) < strings.ToLower(children[j].Author)
		})
	}

	// Пагинация
	total := len(children)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return []entity.Comment{}, total
	}
	if end > total {
		end = total
	}

	return children[start:end], total
}

func (r *CommentRepository) Delete(id string) error {
	// Рекурсивное удаление
	var deleteRecursive func(string) error
	deleteRecursive = func(commentID string) error {
		// Удаляем дочерние комментарии
		childrenKey := fmt.Sprintf("comment:%s:children", commentID)
		childIDs, err := r.client.SMembers(r.ctx, childrenKey).Result()
		if err != nil && err != redis.Nil {
			return err
		}

		for _, childID := range childIDs {
			if err := deleteRecursive(childID); err != nil {
				return err
			}
		}

		// Удаляем из индексов
		comment, exists := r.GetByID(commentID)
		if exists {
			// Удаляем из родительского индекса
			if comment.ParentID == "" {
				r.client.SRem(r.ctx, "comments:root", commentID)
			} else {
				parentKey := fmt.Sprintf("comment:%s:children", comment.ParentID)
				r.client.SRem(r.ctx, parentKey, commentID)
			}

			// Удаляем из поискового индекса
			r.client.SRem(r.ctx, "comments:all", commentID)
			r.removeCommentFromSearchIndex(comment)
		}

		// Удаляем сам комментарий и его children set
		r.client.Del(r.ctx, fmt.Sprintf("comment:%s", commentID))
		r.client.Del(r.ctx, childrenKey)

		return nil
	}

	return deleteRecursive(id)
}

func (r *CommentRepository) Search(query string, page, pageSize int) ([]entity.Comment, int) {
	allComments, err := r.GetAllComments()
	if err != nil {
		return []entity.Comment{}, 0
	}

	var results []entity.Comment
	query = strings.ToLower(query)

	for _, comment := range allComments {
		if strings.Contains(strings.ToLower(comment.Text), query) ||
			strings.Contains(strings.ToLower(comment.Author), query) {
			results = append(results, comment)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	total := len(results)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return []entity.Comment{}, total
	}
	if end > total {
		end = total
	}

	return results[start:end], total
}

func (r *CommentRepository) BuildTree(parentID string, depth int) []entity.Comment {
	if depth > 10 {
		return []entity.Comment{}
	}

	children, _ := r.GetChildren(parentID, 1, 1000, "created_at_asc") // Получаем все дочерние без пагинации
	var tree []entity.Comment

	for _, child := range children {
		node := child
		node.Children = r.BuildTree(node.ID, depth+1)
		tree = append(tree, node)
	}

	return tree
}

func (r *CommentRepository) GetAllComments() ([]entity.Comment, error) {
	ids, err := r.client.SMembers(r.ctx, "comments:all").Result()
	if err != nil {
		return nil, err
	}

	var comments []entity.Comment
	for _, id := range ids {
		if comment, exists := r.GetByID(id); exists {
			comments = append(comments, *comment)
		}
	}

	return comments, nil
}

func (r *CommentRepository) indexCommentForSearch(comment *entity.Comment) error {
	// Индексируем по словам в тексте (упрощенная версия)
	words := strings.Fields(strings.ToLower(comment.Text))
	for _, word := range words {
		if len(word) > 2 { // Игнорируем короткие слова
			key := fmt.Sprintf("search:text:%s", word)
			r.client.SAdd(r.ctx, key, comment.ID)
		}
	}

	// Индексируем по автору
	authorKey := fmt.Sprintf("search:author:%s", strings.ToLower(comment.Author))
	r.client.SAdd(r.ctx, authorKey, comment.ID)

	return nil
}

func (r *CommentRepository) removeCommentFromSearchIndex(comment *entity.Comment) error {
	words := strings.Fields(strings.ToLower(comment.Text))
	for _, word := range words {
		if len(word) > 2 {
			key := fmt.Sprintf("search:text:%s", word)
			r.client.SRem(r.ctx, key, comment.ID)
		}
	}

	authorKey := fmt.Sprintf("search:author:%s", strings.ToLower(comment.Author))
	r.client.SRem(r.ctx, authorKey, comment.ID)

	return nil
}

// Дополнительные методы для управления Redis
func (r *CommentRepository) FlushAll() error {
	return r.client.FlushAll(r.ctx).Err()
}

func (r *CommentRepository) GetStats() (map[string]string, error) {
	stats := make(map[string]string)

	// Количество корневых комментариев
	rootCount, err := r.client.SCard(r.ctx, "comments:root").Result()
	if err != nil {
		return nil, err
	}
	stats["root_comments"] = strconv.FormatInt(rootCount, 10)

	// Общее количество комментариев
	totalCount, err := r.client.SCard(r.ctx, "comments:all").Result()
	if err != nil {
		return nil, err
	}
	stats["total_comments"] = strconv.FormatInt(totalCount, 10)

	// Информация о Redis
	info, err := r.client.Info(r.ctx).Result()
	if err != nil {
		return nil, err
	}
	stats["redis_info"] = info

	return stats, nil
}
