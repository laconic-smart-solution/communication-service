package graphql

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/graphql-go/graphql"

	"github.com/laconic-smart-solution/communication-service/app/normalization-service/internal/client"
)

func facebookPostType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "FacebookPost",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.String},
			"channelId":  &graphql.Field{Type: graphql.String},
			"message":    &graphql.Field{Type: graphql.String},
			"status":     &graphql.Field{Type: graphql.String},
			"externalId": &graphql.Field{Type: graphql.String},
			"platform":   &graphql.Field{Type: graphql.String},
		},
	})
}

func toFacebookPostMap(p client.PostDTO) map[string]interface{} {
	return map[string]interface{}{
		"id":         p.ID.String(),
		"channelId":  p.ChannelID.String(),
		"message":    p.Message,
		"status":     p.Status,
		"externalId": p.ExternalID,
		"platform":   "FACEBOOK",
	}
}

func facebookPostFields(resolver *Resolver, postType *graphql.Object) graphql.Fields {
	return graphql.Fields{
		"posts": &graphql.Field{
			Type: graphql.NewList(postType),
			Args: graphql.FieldConfigArgument{
				"platform":  &graphql.ArgumentConfig{Type: platformEnum()},
				"channelId": &graphql.ArgumentConfig{Type: graphql.String},
				"page":      &graphql.ArgumentConfig{Type: graphql.Int},
				"limit":     &graphql.ArgumentConfig{Type: graphql.Int},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				if p.Args["platform"] != "facebook" {
					return []interface{}{}, nil
				}
				page, limit := 1, 20
				if v, ok := p.Args["page"].(int); ok && v > 0 {
					page = v
				}
				if v, ok := p.Args["limit"].(int); ok && v > 0 {
					limit = v
				}
				channelID, _ := p.Args["channelId"].(string)
				items, err := resolver.Facebook.ListPosts(p.Context, headersFromContext(p.Context), channelID, page, limit)
				if err != nil {
					return nil, err
				}
				out := make([]map[string]interface{}, 0, len(items))
				for _, item := range items {
					out = append(out, toFacebookPostMap(item))
				}
				return out, nil
			},
		},
		"post": &graphql.Field{
			Type: postType,
			Args: graphql.FieldConfigArgument{
				"platform": &graphql.ArgumentConfig{Type: graphql.NewNonNull(platformEnum())},
				"id":       &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				if p.Args["platform"] != "facebook" {
					return nil, nil
				}
				id, err := uuid.Parse(p.Args["id"].(string))
				if err != nil {
					return nil, err
				}
				post, err := resolver.Facebook.GetPost(p.Context, headersFromContext(p.Context), id)
				if err != nil {
					return nil, err
				}
				m := toFacebookPostMap(*post)
				return m, nil
			},
		},
	}
}

func facebookPostMutations(resolver *Resolver, postType *graphql.Object) graphql.Fields {
	createPostInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "CreatePostInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"platform":  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(platformEnum())},
			"channelId": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"message":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	})
	updatePostInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "UpdatePostInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"platform": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(platformEnum())},
			"id":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"message":  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	return graphql.Fields{
		"createPost": &graphql.Field{
			Type: postType,
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createPostInput)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				input, _ := p.Args["input"].(map[string]interface{})
				if input["platform"] != "facebook" {
					return nil, fmt.Errorf("unsupported platform")
				}
				channelID, err := uuid.Parse(input["channelId"].(string))
				if err != nil {
					return nil, err
				}
				post, err := resolver.Facebook.CreatePost(p.Context, headersFromContext(p.Context), channelID, input["message"].(string))
				if err != nil {
					return nil, err
				}
				return toFacebookPostMap(*post), nil
			},
		},
		"updatePost": &graphql.Field{
			Type: postType,
			Args: graphql.FieldConfigArgument{
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updatePostInput)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				input, _ := p.Args["input"].(map[string]interface{})
				if input["platform"] != "facebook" {
					return nil, fmt.Errorf("unsupported platform")
				}
				id, err := uuid.Parse(input["id"].(string))
				if err != nil {
					return nil, err
				}
				post, err := resolver.Facebook.UpdatePost(p.Context, headersFromContext(p.Context), id, input["message"].(string))
				if err != nil {
					return nil, err
				}
				return toFacebookPostMap(*post), nil
			},
		},
		"deletePost": &graphql.Field{
			Type: graphql.Boolean,
			Args: graphql.FieldConfigArgument{
				"platform": &graphql.ArgumentConfig{Type: graphql.NewNonNull(platformEnum())},
				"id":       &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				if p.Args["platform"] != "facebook" {
					return false, fmt.Errorf("unsupported platform")
				}
				id, err := uuid.Parse(p.Args["id"].(string))
				if err != nil {
					return false, err
				}
				if err := resolver.Facebook.DeletePost(p.Context, headersFromContext(p.Context), id); err != nil {
					return false, err
				}
				return true, nil
			},
		},
	}
}

func platformEnum() *graphql.Enum {
	return graphql.NewEnum(graphql.EnumConfig{
		Name: "Platform",
		Values: graphql.EnumValueConfigMap{
			"LINE":     &graphql.EnumValueConfig{Value: "line"},
			"FACEBOOK": &graphql.EnumValueConfig{Value: "facebook"},
		},
	})
}
