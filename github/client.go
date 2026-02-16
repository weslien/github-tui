package github

import (
	"context"

	gogithub "github.com/google/go-github/v68/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

var (
	graphQLClient *githubv4.Client
	restClient    *gogithub.Client
	rateLimiter   *RateLimiter
)

func NewClient(token string) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	graphQLClient = githubv4.NewClient(httpClient)
	restClient = gogithub.NewClient(httpClient)
}

// GetRESTClient returns the initialized REST API client.
func GetRESTClient() *gogithub.Client {
	return restClient
}

// GetGraphQLClient returns the initialized GraphQL API client.
func GetGraphQLClient() *githubv4.Client {
	return graphQLClient
}

func CreateIssue(input githubv4.CreateIssueInput) error {
	var m MutateCreateIssue
	return graphQLClient.Mutate(context.Background(), &m, input, nil)
}

func GetRepos(variables map[string]interface{}) (*Repositories, error) {
	var q struct {
		RepositoryOwner struct {
			Repositories `graphql:"repositories(first: $first, after: $cursor, orderBy: {field: CREATED_AT, direction: DESC})"`
		} `graphql:"repositoryOwner(login: $login)"`
	}

	if err := graphQLClient.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}
	return &q.RepositoryOwner.Repositories, nil
}

func GetRepo(variables map[string]interface{}) (*Repository, error) {
	var q struct {
		Repository `graphql:"repository(owner: $owner, name: $name)"`
	}
	if err := graphQLClient.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}
	return &q.Repository, nil
}

func GetIssues(variables map[string]interface{}) (*Issues, error) {
	var q struct {
		Search Issues `graphql:"search(query: $query, type: ISSUE, first: $first, after: $cursor)"`
	}
	if err := graphQLClient.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}

	issues := &Issues{
		Nodes:    q.Search.Nodes,
		PageInfo: q.Search.PageInfo,
	}
	return issues, nil
}

func GetIssue(variables map[string]interface{}) (*Issue, error) {
	var q struct {
		Repository struct {
			Issue *Issue `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	if err := graphQLClient.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}
	return q.Repository.Issue, nil
}

func GetIssueTemplates(variables map[string]interface{}) ([]IssueTemplate, error) {
	var q struct {
		Repository struct {
			IssueTemplates []IssueTemplate
		} `graphql:"repository(name: $name, owner: $owner)"`
	}
	if err := graphQLClient.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}
	return q.Repository.IssueTemplates, nil
}

func ReopenIssue(id string) error {
	input := githubv4.ReopenIssueInput{
		IssueID: githubv4.String(id),
	}

	var m MutateOpenIsseue

	return graphQLClient.Mutate(context.Background(), &m, input, nil)
}

func CloseIssue(id string) error {
	input := githubv4.CloseIssueInput{
		IssueID: githubv4.String(id),
	}

	var m MutateCoseIssue
	return graphQLClient.Mutate(context.Background(), &m, input, nil)
}

func GetRepoLabels(variables map[string]interface{}) (*Labels, error) {
	var q struct {
		Repository struct {
			Labels `graphql:"labels(first: $first, after: $cursor, orderBy: {field: CREATED_AT, direction: DESC})"`
		} `graphql:"repository(name: $name, owner: $owner)"`
	}
	if err := graphQLClient.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}
	return &q.Repository.Labels, nil
}

func GetRepoMillestones(variables map[string]interface{}) (*Milestones, error) {
	var q struct {
		Repository struct {
			Milestones `graphql:"milestones(first: $first, after: $cursor, orderBy: {field: CREATED_AT, direction: DESC})"`
		} `graphql:"repository(name: $name, owner: $owner)"`
	}
	if err := graphQLClient.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}
	return &q.Repository.Milestones, nil
}

func GetRepoProjects(variables map[string]interface{}) (*Projects, error) {
	var q struct {
		Repository struct {
			Projects `graphql:"projects(first: $first, after: $cursor, orderBy: {field: CREATED_AT, direction: DESC})"`
		} `graphql:"repository(name: $name, owner: $owner)"`
	}
	if err := graphQLClient.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}
	return &q.Repository.Projects, nil
}

func GetRepoAssignableUsers(variables map[string]interface{}) (*AssignableUsers, error) {
	var q struct {
		Repository struct {
			AssignableUsers `graphql:"assignableUsers(first: $first, after: $cursor)"`
		} `graphql:"repository(name: $name, owner: $owner)"`
	}
	if err := graphQLClient.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}
	return &q.Repository.AssignableUsers, nil
}

func DeleteIssueComment(id string) error {
	var m MutateDeleteComment
	input := githubv4.DeleteIssueCommentInput{
		ID: githubv4.ID(id),
	}
	return graphQLClient.Mutate(context.Background(), &m, input, nil)
}

func UpdateIssue(input githubv4.UpdateIssueInput) error {
	var m MutateUpdateIssue
	return graphQLClient.Mutate(context.Background(), &m, input, nil)
}

func UpdateIssueComment(input githubv4.UpdateIssueCommentInput) error {
	var m MutateUpdateIssueComment
	return graphQLClient.Mutate(context.Background(), &m, input, nil)
}

func AddIssueComment(input githubv4.AddCommentInput) error {
	var m MutateAddIssueComment
	return graphQLClient.Mutate(context.Background(), &m, input, nil)
}
