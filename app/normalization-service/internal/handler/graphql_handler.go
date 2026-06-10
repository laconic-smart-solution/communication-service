package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/graphql-go/graphql"

	gql "github.com/laconic-smart-solution/communication-service/app/normalization-service/internal/graphql"
)

type GraphQLHandler struct {
	schema graphql.Schema
}

func NewGraphQLHandler(schema graphql.Schema) *GraphQLHandler {
	return &GraphQLHandler{schema: schema}
}

func (h *GraphQLHandler) Handle(c *fiber.Ctx) error {
	var body struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": []fiber.Map{{"message": "invalid json body"}}})
	}

	headers := map[string]string{
		"X-Tenant-ID":      c.Get("X-Tenant-ID"),
		"X-Actor-ID":       firstNonEmpty(c.Get("X-Actor-ID"), c.Get("X-User-ID")),
		"X-Gateway-Secret": c.Get("X-Gateway-Secret"),
		"X-Request-ID":     c.Get("X-Request-ID"),
	}

	ctx := gql.WithHeaders(c.Context(), headers)
	result := graphql.Do(graphql.Params{
		Schema:         h.schema,
		RequestString:  body.Query,
		VariableValues: body.Variables,
		OperationName:  body.OperationName,
		Context:        ctx,
	})

	return c.Status(fiber.StatusOK).JSON(result)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (h *GraphQLHandler) Playground(c *fiber.Ctx) error {
	html := `<!DOCTYPE html>
<html><head><title>GraphQL Playground</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css"/>
<script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head><body>
<div id="root"></div>
<script>GraphQLPlayground.init(document.getElementById('root'), { endpoint: '/graphql' })</script>
</body></html>`
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}
