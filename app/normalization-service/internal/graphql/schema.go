package graphql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/graphql-go/graphql"

	"github.com/laconic-smart-solution/communication-service/app/normalization-service/internal/client"
)

type Resolver struct {
	Line     client.LineClient
	Facebook client.FacebookClient
}

func NewSchema(resolver *Resolver) (graphql.Schema, error) {
	platformEnum := platformEnum()
	postType := facebookPostType()

	channelType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Channel",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.String},
			"externalId":  &graphql.Field{Type: graphql.String},
			"displayName": &graphql.Field{Type: graphql.String},
			"platform":    &graphql.Field{Type: graphql.String},
		},
	})

	messageType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Message",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.String},
			"channelId":   &graphql.Field{Type: graphql.String},
			"recipientId": &graphql.Field{Type: graphql.String},
			"body":        &graphql.Field{Type: graphql.String},
			"status":      &graphql.Field{Type: graphql.String},
			"platform":    &graphql.Field{Type: graphql.String},
		},
	})

	sendInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "SendMessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"platform":    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(platformEnum)},
			"channelId":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"recipientId": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"text":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	connectInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "ConnectChannelInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"platform":    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(platformEnum)},
			"externalId":  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"displayName": &graphql.InputObjectFieldConfig{Type: graphql.String},
			"accessToken": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	queryFields := graphql.Fields{
		"channels": &graphql.Field{
			Type: graphql.NewList(channelType),
			Args: graphql.FieldConfigArgument{
				"platform": &graphql.ArgumentConfig{Type: platformEnum},
				"page":     &graphql.ArgumentConfig{Type: graphql.Int},
				"limit":    &graphql.ArgumentConfig{Type: graphql.Int},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				platform, _ := p.Args["platform"].(string)
				page, limit := 1, 20
				if v, ok := p.Args["page"].(int); ok && v > 0 {
					page = v
				}
				if v, ok := p.Args["limit"].(int); ok && v > 0 {
					limit = v
				}
				headers := headersFromContext(p.Context)
				out := make([]map[string]interface{}, 0)

				if platform == "" || platform == "line" {
					items, err := resolver.Line.ListChannels(p.Context, headers, page, limit)
					if err != nil {
						return nil, err
					}
					for _, ch := range items {
						out = append(out, map[string]interface{}{
							"id": ch.ID.String(), "externalId": ch.ExternalID,
							"displayName": ch.DisplayName, "platform": "LINE",
						})
					}
				}
				if platform == "" || platform == "facebook" {
					items, err := resolver.Facebook.ListChannels(p.Context, headers, page, limit)
					if err != nil {
						return nil, err
					}
					for _, ch := range items {
						out = append(out, map[string]interface{}{
							"id": ch.ID.String(), "externalId": ch.ExternalID,
							"displayName": ch.DisplayName, "platform": "FACEBOOK",
						})
					}
				}
				return out, nil
			},
		},
		"message": &graphql.Field{
			Type: messageType,
			Args: graphql.FieldConfigArgument{
				"platform": &graphql.ArgumentConfig{Type: graphql.NewNonNull(platformEnum)},
				"id":       &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				platform, _ := p.Args["platform"].(string)
				if platform != "line" {
					return nil, nil
				}
				id, err := uuid.Parse(p.Args["id"].(string))
				if err != nil {
					return nil, err
				}
				msg, err := resolver.Line.GetMessage(p.Context, headersFromContext(p.Context), id)
				if err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"id": msg.ID.String(), "channelId": msg.ChannelID.String(),
					"recipientId": msg.RecipientID, "body": msg.Body,
					"status": msg.Status, "platform": "LINE",
				}, nil
			},
		},
	}

	for k, v := range facebookPostFields(resolver, postType) {
		queryFields[k] = v
	}

	mutationFields := graphql.Fields{
		"sendMessage": &graphql.Field{
			Type: messageType,
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(sendInput)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				input, _ := p.Args["input"].(map[string]interface{})
				if input["platform"] != "line" {
					return nil, fmt.Errorf("unsupported platform")
				}
				channelID, err := uuid.Parse(input["channelId"].(string))
				if err != nil {
					return nil, err
				}
				msg, err := resolver.Line.SendMessage(p.Context, headersFromContext(p.Context), channelID, input["recipientId"].(string), input["text"].(string))
				if err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"id": msg.ID.String(), "channelId": msg.ChannelID.String(),
					"recipientId": msg.RecipientID, "body": msg.Body,
					"status": msg.Status, "platform": "LINE",
				}, nil
			},
		},
		"connectChannel": &graphql.Field{
			Type: channelType,
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(connectInput)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				input, _ := p.Args["input"].(map[string]interface{})
				platform := input["platform"].(string)
				displayName, _ := input["displayName"].(string)
				headers := headersFromContext(p.Context)
				switch platform {
				case "line":
					ch, err := resolver.Line.ConnectChannel(p.Context, headers, input["externalId"].(string), displayName, input["accessToken"].(string))
					if err != nil {
						return nil, err
					}
					return map[string]interface{}{
						"id": ch.ID.String(), "externalId": ch.ExternalID,
						"displayName": ch.DisplayName, "platform": "LINE",
					}, nil
				case "facebook":
					ch, err := resolver.Facebook.ConnectChannel(p.Context, headers, input["externalId"].(string), displayName, input["accessToken"].(string))
					if err != nil {
						return nil, err
					}
					return map[string]interface{}{
						"id": ch.ID.String(), "externalId": ch.ExternalID,
						"displayName": ch.DisplayName, "platform": "FACEBOOK",
					}, nil
				default:
					return nil, fmt.Errorf("unsupported platform")
				}
			},
		},
		"revokeChannel": &graphql.Field{
			Type: graphql.Boolean,
			Args: graphql.FieldConfigArgument{
				"platform": &graphql.ArgumentConfig{Type: graphql.NewNonNull(platformEnum)},
				"id":       &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				platform := p.Args["platform"].(string)
				id, err := uuid.Parse(p.Args["id"].(string))
				if err != nil {
					return false, err
				}
				headers := headersFromContext(p.Context)
				switch platform {
				case "line":
					return true, resolver.Line.RevokeChannel(p.Context, headers, id)
				case "facebook":
					return true, resolver.Facebook.RevokeChannel(p.Context, headers, id)
				default:
					return false, fmt.Errorf("unsupported platform")
				}
			},
		},
	}

	for k, v := range facebookPostMutations(resolver, postType) {
		mutationFields[k] = v
	}

	query := graphql.NewObject(graphql.ObjectConfig{Name: "Query", Fields: queryFields})
	mutation := graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: mutationFields})

	return graphql.NewSchema(graphql.SchemaConfig{Query: query, Mutation: mutation})
}

type ctxKey string

const headersKey ctxKey = "forwardHeaders"

func WithHeaders(ctx context.Context, headers map[string]string) context.Context {
	return context.WithValue(ctx, headersKey, headers)
}

func headersFromContext(ctx context.Context) map[string]string {
	if v, ok := ctx.Value(headersKey).(map[string]string); ok {
		return v
	}
	return map[string]string{}
}
