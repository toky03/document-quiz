package main

import (
	"context"
	"fmt"
	"os"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	chromaembeddings "github.com/amikos-tech/chroma-go/pkg/embeddings"
	chromaopenai "github.com/amikos-tech/chroma-go/pkg/embeddings/openai"
)

type VectorChunk struct {
	ID         string
	SourceName string
	ChunkIndex int
	ChunkText  string
}

func initVectorStoreDB() error {
	return os.MkdirAll(VectorDBDir, 0o755)
}

func replaceVectorChunks(sourceName string, chunks []VectorChunk, openAIAPIKey string) error {
	embedder, err := chromaopenai.NewOpenAIEmbeddingFunction(
		openAIAPIKey,
		chromaopenai.WithModel(chromaopenai.TextEmbedding3Small),
	)
	if err != nil {
		return err
	}

	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}

	client, err := chroma.NewHTTPClient(
		chroma.WithBaseURL(chromaURL),
	)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	collection, err := client.GetOrCreateCollection(
		context.Background(),
		CollectionName,
		chroma.WithEmbeddingFunctionCreate(embedder),
		chroma.WithHNSWSpaceCreate(chromaembeddings.COSINE),
	)
	if err != nil {
		return fmt.Errorf("chroma Collection konnte nicht geladen werden: %w", err)
	}
	defer func() { _ = collection.Close() }()

	// Remove only vectors for this source and keep all other existing data in vector_db.
	err = collection.Delete(
		context.Background(),
		chroma.WithWhereDelete(chroma.EqString("source_name", sourceName)),
	)
	if err != nil {
		return fmt.Errorf("chroma Delete für Quelle fehlgeschlagen: %w", err)
	}

	if len(chunks) == 0 {
		return nil
	}

	ids := make([]chroma.DocumentID, 0, len(chunks))
	metadatas := make([]chroma.DocumentMetadata, 0, len(chunks))
	documents := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		ids = append(ids, chroma.DocumentID(chunk.ID))
		documents = append(documents, chunk.ChunkText)
		metadata, metadataErr := chroma.NewDocumentMetadataFromMap(map[string]interface{}{
			"source_name": chunk.SourceName,
			"chunk_index": chunk.ChunkIndex,
			"chunk_id":    chunk.ID,
		})
		if metadataErr != nil {
			return fmt.Errorf(
				"chroma Dokument-Metadaten konnten nicht erstellt werden: %w",
				metadataErr,
			)
		}
		metadatas = append(metadatas, metadata)
	}

	if err = collection.Upsert(
		context.Background(),
		chroma.WithIDs(ids...),
		chroma.WithMetadatas(metadatas...),
		chroma.WithTexts(documents...),
	); err != nil {
		return fmt.Errorf("chroma Upsert fehlgeschlagen: %w", err)
	}

	return nil
}
