package main

import (
	"context"
	"github.com/samsarahq/thunder/batch"
	"net/http"
	"strconv"
	"time"

	"github.com/samsarahq/thunder/graphql"
	"github.com/samsarahq/thunder/graphql/graphiql"
	"github.com/samsarahq/thunder/graphql/introspection"
	"github.com/samsarahq/thunder/graphql/schemabuilder"
	"github.com/samsarahq/thunder/reactive"
)

type post struct {
	Id        string
	Title     string
	Body      string
	CreatedAt time.Time
	Author 	  string
}


type postArg struct {
	Title     string
	Body      string
	Author 	  string
}

type Author struct {
	Id string
	Name string
}

// server is our graphql server.
type server struct {
	posts []post
	authors []Author
}

// registerQuery registers the root query type.
func (s *server) registerQuery(schema *schemabuilder.Schema) {
	obj := schema.Query()

	obj.FieldFunc("posts", func() []post {
		return s.posts
	})
	obj.FieldFunc("authors", func() []Author {
		return s.authors
	})
}

// registerMutation registers the root mutation type.
func (s *server) registerMutation(schema *schemabuilder.Schema) {
	obj := schema.Mutation()
	obj.FieldFunc("echo", func(args struct{ Message string }) string {
		return args.Message + "!!"
	})

	obj.FieldFunc("addPost", func(args postArg) string {
		s.posts = append(s.posts, post{
			Id : "p" + strconv.FormatInt(time.Now().Unix(), 10),
			Title: args.Title,
			Body: args.Body,
			Author: args.Author,
			CreatedAt: time.Now(),
		})
		return args.Title + "!!"
	})
}

// registerPost registers the post type.
func (s *server) registerPost(schema *schemabuilder.Schema) {
	obj := schema.Object("Post", post{})
	obj.FieldFunc("key", func(ctx context.Context, p *post) string {
		return "key-" + p.Id
	})
	obj.FieldFunc("age", func(ctx context.Context, p *post) string {
		reactive.InvalidateAfter(ctx, 5*time.Second)
		return time.Since(p.CreatedAt).String()
	})

	obj.BatchFieldFunc("author", func(ctx context.Context, in map[batch.Index]*post) (map[batch.Index]*Author, error) {
		authorMap := make(map[string]Author,0)
		for _, author := range s.authors{
			authorMap[author.Id] = author
		}
		out := make(map[batch.Index]*Author)
		for i, foo := range in {
			author := authorMap[foo.Author]
			out[i] = &author
		}
		return out, nil
	})
}

// schema builds the graphql schema.
func (s *server) schema() *graphql.Schema {
	builder := schemabuilder.NewSchema()
	s.registerQuery(builder)
	s.registerMutation(builder)
	s.registerPost(builder)
	return builder.MustBuild()
}

func main() {
	// Instantiate a server, build a server, and serve the schema on port 3030.
	server := &server{
		posts: []post{
			{Id:"p1", Title: "first post!", Body: "I was here first!", CreatedAt: time.Now(), Author:"id123"},
			{Id:"p2", Title: "graphql", Body: "did you hear about Thunder?", CreatedAt: time.Now(), Author: "id789"},
		},
		authors: []Author{
			{Id:"id123", Name: "Foo Cai"},
			{Id:"id456", Name: "Bar Li"},
			{Id:"id789", Name: "Lives Zhao"},
		},
	}

	schema := server.schema()
	introspection.AddIntrospectionToSchema(schema)

	// Expose schema and graphiql.
	http.Handle("/graphql", graphql.HTTPHandler(schema))
	http.Handle("/graphiql/", http.StripPrefix("/graphiql/", graphiql.Handler()))
	http.ListenAndServe(":3030", nil)
}