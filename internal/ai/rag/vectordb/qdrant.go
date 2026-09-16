package vectordb

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"

	"agent-desk/internal/pkg/config"
)

type QdrantProvider struct {
	client *qdrant.Client
}

func NewQdrantProvider(cfg *config.QdrantVectorDBConfig) (*QdrantProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("vectordb config is nil")
	}

	host := cfg.Host
	if host == "" {
		host = "localhost"
	}

	port := cfg.GrpcPort
	if port <= 0 {
		port = 6334
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   host,
		Port:                   port,
		APIKey:                 cfg.APIKey,
		UseTLS:                 cfg.UseTLS,
		SkipCompatibilityCheck: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	return &QdrantProvider{client: client}, nil
}

func (p *QdrantProvider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

func (p *QdrantProvider) CreateCollection(ctx context.Context, name string, dimension int) error {
	err := p.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(dimension),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", name, err)
	}
	return nil
}

func (p *QdrantProvider) DeleteCollection(ctx context.Context, name string) error {
	err := p.client.DeleteCollection(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete collection %s: %w", name, err)
	}
	return nil
}

func (p *QdrantProvider) GetCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	info, err := p.client.GetCollectionInfo(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection %s: %w", name, err)
	}

	status := info.GetStatus().String()
	pointCount := int(info.GetPointsCount())

	dimension := 0
	if info.Config != nil && info.Config.Params != nil {
		vectorsConfig := info.Config.Params.VectorsConfig
		if vectorsConfig != nil {
			params := vectorsConfig.GetParams()
			if params != nil {
				dimension = int(params.Size)
			}
		}
	}

	return &CollectionInfo{
		Name:       name,
		Dimension:  dimension,
		PointCount: pointCount,
		Status:     status,
	}, nil
}

func (p *QdrantProvider) ListCollections(ctx context.Context) ([]string, error) {
	collections, err := p.client.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	return collections, nil
}

func (p *QdrantProvider) UpsertVectors(ctx context.Context, collectionName string, vectors []Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, 0, len(vectors))
	for _, v := range vectors {
		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewID(v.ID),
			Vectors: qdrant.NewVectors(v.Vector...),
			Payload: qdrant.NewValueMap(v.Payload.ToMap()),
		})
	}

	_, err := p.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert vectors to collection %s: %w", collectionName, err)
	}
	return nil
}

func (p *QdrantProvider) DeleteVectors(ctx context.Context, collectionName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	pointIDs := make([]*qdrant.PointId, 0, len(ids))
	for _, id := range ids {
		pointIDs = append(pointIDs, qdrant.NewID(id))
	}

	_, err := p.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: pointIDs,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete vectors from collection %s: %w", collectionName, err)
	}
	return nil
}

func (p *QdrantProvider) Search(ctx context.Context, req *SearchRequest) ([]SearchResult, error) {
	filter := p.buildFilter(req.Filter)

	results, err := p.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: req.CollectionName,
		Query:          qdrant.NewQuery(req.Vector...),
		Limit:          qdrant.PtrOf(uint64(req.TopK)),
		ScoreThreshold: &req.ScoreThreshold,
		Filter:         filter,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search collection %s: %w", req.CollectionName, err)
	}

	searchResults := make([]SearchResult, 0, len(results))
	for _, r := range results {
		payload := make(map[string]any)
		if r.Payload != nil {
			for k, v := range r.Payload {
				payload[k] = p.extractPayloadValue(v)
			}
		}

		id := ""
		if r.Id != nil {
			id = r.Id.GetUuid()
		}

		searchResults = append(searchResults, SearchResult{
			ID:      id,
			Score:   r.Score,
			Payload: ChunkPayloadFromMap(payload),
		})
	}

	return searchResults, nil
}

func (p *QdrantProvider) buildFilter(filter *SearchFilter) *qdrant.Filter {
	if filter == nil {
		return nil
	}

	must := make([]*qdrant.Condition, 0, 2)
	if len(filter.KnowledgeBaseIDs) > 0 {
		must = append(must, qdrant.NewMatchInts("knowledge_base_id", filter.KnowledgeBaseIDs...))
	}
	if len(filter.DocumentIDs) > 0 {
		must = append(must, qdrant.NewMatchInts("document_id", filter.DocumentIDs...))
	}
	if len(must) == 0 {
		return nil
	}

	return &qdrant.Filter{Must: must}
}

func (p *QdrantProvider) extractPayloadValue(v *qdrant.Value) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.Kind.(type) {
	case *qdrant.Value_StringValue:
		return val.StringValue
	case *qdrant.Value_IntegerValue:
		return val.IntegerValue
	case *qdrant.Value_DoubleValue:
		return val.DoubleValue
	case *qdrant.Value_BoolValue:
		return val.BoolValue
	case *qdrant.Value_ListValue:
		list := make([]interface{}, 0, len(val.ListValue.Values))
		for _, item := range val.ListValue.Values {
			list = append(list, p.extractPayloadValue(item))
		}
		return list
	case *qdrant.Value_StructValue:
		m := make(map[string]interface{})
		for k, v := range val.StructValue.Fields {
			m[k] = p.extractPayloadValue(v)
		}
		return m
	default:
		return nil
	}
}
