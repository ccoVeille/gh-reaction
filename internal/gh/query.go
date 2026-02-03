package gh

import (
	"context"
	_ "embed" // for embedding GraphQL queries
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/ccoVeille/gh-reaction/internal/gh/queries"
	"github.com/ccoVeille/gh-reaction/internal/spinner"
	"github.com/ccoVeille/gh-reaction/internal/timeago"
	"github.com/cli/go-gh/v2/pkg/api"
)

type PageInfo struct {
	HasNextPage     bool   `json:"hasNextPage,omitempty"`
	EndCursor       string `json:"endCursor,omitempty"`
	HasPreviousPage bool   `json:"hasPreviousPage,omitempty"`
	StartCursor     string `json:"startCursor,omitempty"`
}

func (p PageInfo) IsZero() bool {
	return !p.HasNextPage && !p.HasPreviousPage
}

type Reaction struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	User      struct {
		Type     string `json:"__typename"`
		IsViewer bool   `json:"isViewer"`
		Login    string `json:"login"`
		Name     string `json:"name,omitempty"`
	} `json:"user"`
}

type Reactions struct {
	TotalCount int        `json:"totalCount,omitempty"`
	PageInfo   PageInfo   `json:"pageInfo,omitzero"`
	Nodes      []Reaction `json:"nodes,omitempty"`
}

type Node struct {
	NodeMetadata
	Reactions Reactions `json:"reactions"`
}

type NodeMetadata struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	Body   Body   `json:"body,omitempty"`
	Author struct {
		Login    string `json:"login"`
		Name     string `json:"name,omitempty"`
		Type     string `json:"__typename"`
		IsViewer bool   `json:"isViewer"`
	} `json:"author,omitempty"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	PullRequest *struct {
		ID string `json:"id"`
	} `json:"pullRequest,omitempty"`
	// TODO: use this to fetch the role of the people reacting to the message
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository,omitempty"`
	Comments struct {
		TotalCount int `json:"totalCount,omitempty"`
	} `json:"comments,omitempty"`
}

type ObjectType string

const (
	ObjectTypePullRequest        ObjectType = "pull_request"
	ObjectTypePullRequestComment ObjectType = "pull_request_comment"
	ObjectTypeIssue              ObjectType = "issue"
	ObjectTypeIssueComment       ObjectType = "issue_comment"
	ObjectTypeDiscussion         ObjectType = "discussion"
	ObjectTypeDiscussionComment  ObjectType = "discussion_comment"
	ObjectTypeCommitComment      ObjectType = "commit_comment"
)

type resReaction struct {
	Reaction
	Parent NodeMetadata `json:"parent"`
	Type   ObjectType   `json:"type"`
}

// ReactionResult exposes reaction data for consumers without exporting internals.
type ReactionResult = resReaction

type resNode struct {
	TotalCount int      `json:"totalCount,omitempty"`
	PageInfo   PageInfo `json:"pageInfo,omitzero"`
	Nodes      []Node   `json:"nodes,omitempty"`
}

type Body string

func (b Body) MarshalJSON() ([]byte, error) {
	if len(b) > 20 {
		b = b[:20] + "... (truncated)"
	}

	return json.Marshal(string(b))
}

type resAPISingle struct {
	User struct {
		CommitComments               resNode `json:"commitComments,omitzero"`
		IssueComments                resNode `json:"issueComments,omitzero"`
		Issues                       resNode `json:"issues,omitzero"`
		PullRequests                 resNode `json:"pullRequests,omitzero"`
		RepositoryDiscussionComments resNode `json:"repositoryDiscussionComments,omitzero"`
		RepositoryDiscussions        resNode `json:"repositoryDiscussions,omitzero"`
	} `json:"user"`
	Repository struct {
		Issues        resNode `json:"issues,omitzero"`
		IssueComments resNode `json:"issueComments,omitzero"`
		PullRequests  resNode `json:"pullRequests,omitzero"`
		Discussions   resNode `json:"discussions,omitzero"`
	} `json:"repository"`
}

func (r *resAPISingle) extractByObjectType(requestType string, objType ObjectType) resNode {
	if requestType == "repository" {
		switch objType {
		case ObjectTypePullRequest:
			return r.Repository.PullRequests
		case ObjectTypeIssue:
			return r.Repository.Issues
		case ObjectTypeIssueComment:
			return r.Repository.IssueComments
		case ObjectTypeDiscussion:
			return r.Repository.Discussions
		}
	}
	if requestType == "user" {
		switch objType {
		case ObjectTypePullRequest:
			return r.User.PullRequests
		case ObjectTypeIssue:
			return r.User.Issues
		case ObjectTypeIssueComment:
			return r.User.IssueComments
		case ObjectTypeDiscussion:
			return r.User.RepositoryDiscussions
		case ObjectTypeDiscussionComment:
			return r.User.RepositoryDiscussionComments
		case ObjectTypeCommitComment:
			return r.User.CommitComments
		}
	}
	return resNode{}
}

type resAPINodeReactions struct {
	Node struct {
		Reactions Reactions `json:"reactions"`
	} `json:"node"`
}

type resAPIRepoIssueComments struct {
	Repository struct {
		Issues struct {
			TotalCount int      `json:"totalCount,omitempty"`
			PageInfo   PageInfo `json:"pageInfo,omitzero"`
			Nodes      []struct {
				Comments resNode `json:"comments"`
			} `json:"nodes,omitempty"`
		} `json:"issues"`
	} `json:"repository"`
}

type resAPIRepoDiscussionComments struct {
	Repository struct {
		Discussions struct {
			TotalCount int      `json:"totalCount,omitempty"`
			PageInfo   PageInfo `json:"pageInfo,omitzero"`
			Nodes      []struct {
				Comments resNode `json:"comments"`
			} `json:"nodes,omitempty"`
		} `json:"discussions"`
	} `json:"repository"`
}

type requester struct {
	client  *api.GraphQLClient
	author  string
	minDate timeago.RelativeDate
	spin    *spinner.Spinner
}

// QueryUser fetches messages posted by the user using separate queries per post type
func QueryUser(ctx context.Context, minDate timeago.RelativeDate, author string) ([]ReactionResult, error) {
	clientGraphQL, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}

	suffix := fmt.Sprintf("on github.com since %s", minDate.String())
	fmt.Printf("Looking for posts %s\n", suffix)

	spin := spinner.New(os.Stdout)
	spin.Start(ctx, "fetching posts")

	req := &requester{
		client:  clientGraphQL,
		author:  author,
		minDate: minDate,
		spin:    spin,
	}

	// Define post type configurations
	postTypes := map[ObjectType]string{
		ObjectTypePullRequest:       queries.AuthorPullRequests,
		ObjectTypeIssue:             queries.AuthorIssues,
		ObjectTypeIssueComment:      queries.AuthorIssueComments,
		ObjectTypeDiscussion:        queries.AuthorRepositoryDiscussions,
		ObjectTypeDiscussionComment: queries.AuthorRepositoryDiscussionComments,
		ObjectTypeCommitComment:     queries.AuthorCommitComments,
	}

	type result struct {
		reactions []resReaction
	}

	var wg sync.WaitGroup

	errCh := make(chan error, len(postTypes))
	resultCh := make(chan result, len(postTypes))

	// Process each post type in parallel
	for objType, query := range postTypes {
		wg.Go(func() {
			reactions, err := req.fetchUserPostsReactions(
				ctx,
				objType,
				query,
			)
			if err != nil {
				errCh <- fmt.Errorf("%s: %w", objType, err)
				return
			}

			resultCh <- result{reactions: reactions}
		})
	}

	// Wait for all goroutines and close channels
	wg.Wait()
	close(errCh)
	close(resultCh)

	var errs []error
	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("errors fetching reactions: %v", errs)
	}

	// Collect results
	var allReactions []resReaction
	for res := range resultCh {
		allReactions = append(allReactions, res.reactions...)
	}

	posts, reactions := spin.GetCounts()
	spin.Done("fetched %d posts and %d reactions in total", posts, reactions)

	// Sort reactions by creation time
	slices.SortFunc(allReactions, func(a, b resReaction) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return 0
	})

	return allReactions, nil
}

func (r *requester) fetchUserPostsReactions(
	ctx context.Context,
	objectType ObjectType,
	query string,
) ([]resReaction, error) {
	args := map[string]any{
		"login":         r.author,
		"last_reaction": 5, // most node has less than 5 reactions
		"nb":            100,
	}

	var reactions []resReaction

	for {
		var res resAPISingle

		err := r.client.DoWithContext(ctx, query, args, &res)
		if err != nil {
			return nil, err
		}

		nodes := res.extractByObjectType("user", objectType)
		var hasRecentPosts bool

		// Extract reactions from nodes
		for _, node := range nodes.Nodes {
			// Update progress display
			posts, reacts := r.spin.GetCounts()
			r.spin.Progress("fetched %d posts and %d reactions", posts, reacts)

			if node.CreatedAt.Before(r.minDate.Time) {
				// a recent reaction on an old post will be ignored
				continue
			}
			hasRecentPosts = true

			r.spin.AddPosts(1)

			reactionNodes := node.Reactions.Nodes

			if node.Reactions.PageInfo.HasPreviousPage {
				// Fetch all reactions for this node if there are more pages
				moreReactionNodes, err := r.fetchAllReactionsForNode(
					ctx,
					node.ID,
					node.Reactions.PageInfo.StartCursor, // we start from the current cursor
				)
				if err != nil {
					return nil, err
				}
				reactionNodes = append(reactionNodes, moreReactionNodes...)
			}

			// Adjust type for issue comments on PRs
			objType := objectType
			if objectType == ObjectTypeIssueComment && node.PullRequest != nil {
				objType = ObjectTypePullRequestComment
			}

			for _, react := range reactionNodes {
				reactions = append(reactions, resReaction{
					Reaction: react,
					Parent:   node.NodeMetadata,
					Type:     objType,
				})
				r.spin.AddReactions(1)
			}
		}

		if !hasRecentPosts {
			// no recent posts in this batch, we can stop the next pages will also be older
			break
		}

		if !nodes.PageInfo.HasPreviousPage || nodes.PageInfo.StartCursor == "" {
			// no more pages
			break
		}

		args["cursor"] = nodes.PageInfo.StartCursor
	}

	return reactions, nil
}

func (r *requester) fetchAllReactionsForNode(
	ctx context.Context,
	nodeID string,
	startCursor string,
) ([]Reaction, error) {
	if startCursor == "" {
		return nil, nil
	}

	args := map[string]any{
		"id":            nodeID,
		"last_reaction": 100,
		"cursor":        startCursor,
	}

	var all []Reaction
	for {
		var res resAPINodeReactions
		if err := r.client.DoWithContext(ctx, queries.AuthorReactionsByNode, args, &res); err != nil {
			return nil, err
		}

		reactions := res.Node.Reactions
		if len(reactions.Nodes) == 0 {
			break
		}

		all = append(all, reactions.Nodes...)
		if !reactions.PageInfo.HasPreviousPage {
			break
		}

		args["cursor"] = reactions.PageInfo.StartCursor
	}

	return all, nil
}

// QueryRepository fetches reactions on posts in a specific repository
func QueryRepository(ctx context.Context, minDate timeago.RelativeDate, repo Repository) ([]ReactionResult, error) {
	clientGraphQL, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}

	suffix := fmt.Sprintf("in %s/%s since %s", repo.Owner, repo.Name, minDate.String())
	fmt.Printf("Looking for repository reactions %s\n", suffix)

	spin := spinner.New(os.Stdout)
	spin.Start(ctx, "fetching repository items")

	req := &requester{
		client:  clientGraphQL,
		minDate: minDate,
		spin:    spin,
	}

	// Define post type configurations for repository
	postTypes := map[ObjectType]string{
		ObjectTypePullRequest:       queries.RepositoryPullRequests,
		ObjectTypeIssue:             queries.RepositoryIssues,
		ObjectTypeDiscussion:        queries.RepositoryDiscussions,
		ObjectTypeDiscussionComment: queries.RepositoryDiscussionComments,
		ObjectTypeIssueComment:      queries.RepositoryIssueComments,
	}

	errCh := make(chan error, len(postTypes))
	resultCh := make(chan []resReaction, len(postTypes))

	var wg sync.WaitGroup
	for objType, query := range postTypes {
		wg.Go(func() {
			var reactions []resReaction
			var err error
			switch objType {
			case ObjectTypeIssueComment:
				reactions, err = req.fetchRepositoryIssueCommentsReactions(
					ctx,
					repo,
					query,
				)
			case ObjectTypeDiscussionComment:
				reactions, err = req.fetchRepositoryDiscussionCommentsReactions(
					ctx,
					repo,
					query,
				)
			default:
				reactions, err = req.fetchRepositoryItemsReactions(
					ctx,
					objType,
					repo,
					query,
				)
			}
			if err != nil {
				errCh <- fmt.Errorf("%s: %w", objType, err)
				return
			}

			resultCh <- reactions
		})
	}

	// Wait for all goroutines and close channels
	wg.Wait()
	close(errCh)
	close(resultCh)

	var errs []error
	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("errors fetching reactions: %v", errs)
	}

	// Collect results
	var allReactions []resReaction
	for res := range resultCh {
		allReactions = append(allReactions, res...)
	}

	posts, reactions := spin.GetCounts()
	spin.Done("fetched %d items and %d reactions in total", posts, reactions)

	// Sort reactions by creation time
	slices.SortFunc(allReactions, func(a, b resReaction) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return 0
	})

	return allReactions, nil
}

func (r *requester) fetchRepositoryItemsReactions(
	ctx context.Context,
	objectType ObjectType,
	repo Repository,
	query string,
) ([]resReaction, error) {
	args := map[string]any{
		"owner":         repo.Owner,
		"name":          repo.Name,
		"last_reaction": 5, // most node has less than 5 reactions
		"nb":            50,
	}

	var reactions []resReaction

	for {

		var res resAPISingle
		err := r.client.DoWithContext(ctx, query, args, &res)
		if err != nil {
			return nil, err
		}

		nodes := res.extractByObjectType("repository", objectType)
		var hasRecentPosts bool

		// Extract reactions from nodes
		for _, node := range nodes.Nodes {
			if node.CreatedAt.Before(r.minDate.Time) {
				continue
			}
			hasRecentPosts = true

			r.spin.AddPosts(1)

			reactionNodes := node.Reactions.Nodes

			if node.Reactions.PageInfo.HasPreviousPage {
				// Fetch all reactions for this node if there are more pages
				moreReactionNodes, err := r.fetchAllReactionsForNode(
					ctx,
					node.ID,
					node.Reactions.PageInfo.StartCursor, // we start from the current cursor
				)
				if err != nil {
					return nil, err
				}
				reactionNodes = append(reactionNodes, moreReactionNodes...)
			}

			// Adjust type for issue comments on PRs
			objType := objectType
			if objectType == ObjectTypeIssueComment && node.PullRequest != nil {
				objType = ObjectTypePullRequestComment
			}

			for _, react := range reactionNodes {
				if react.CreatedAt.Before(r.minDate.Time) {
					continue
				}

				reactions = append(reactions, resReaction{
					Reaction: react,
					Parent:   node.NodeMetadata,
					Type:     objType,
				})
				r.spin.AddReactions(1)
			}

			// Update progress display
			posts, reacts := r.spin.GetCounts()
			r.spin.Progress("fetched %d items and %d reactions", posts, reacts)
		}

		if !hasRecentPosts {
			break
		}

		// Handle both forward (hasNextPage/endCursor) and backward (hasPreviousPage/startCursor) pagination
		if nodes.PageInfo.HasNextPage && nodes.PageInfo.EndCursor != "" {
			args["cursor"] = nodes.PageInfo.EndCursor
		} else if nodes.PageInfo.HasPreviousPage && nodes.PageInfo.StartCursor != "" {
			args["cursor"] = nodes.PageInfo.StartCursor
		} else {
			// no more pages
			break
		}
	}

	return reactions, nil
}

func (r *requester) fetchRepositoryIssueCommentsReactions(
	ctx context.Context,
	repo Repository,
	query string,
) ([]resReaction, error) {
	args := map[string]any{
		"owner":         repo.Owner,
		"name":          repo.Name,
		"last_reaction": 5, // most node has less than 5 reactions
		"nb":            50,
	}

	var reactions []resReaction

	for {
		var res resAPIRepoIssueComments
		err := r.client.DoWithContext(ctx, query, args, &res)
		if err != nil {
			return nil, err
		}

		issues := res.Repository.Issues
		var hasRecentPosts bool

		for _, issue := range issues.Nodes {
			comments := issue.Comments
			for _, node := range comments.Nodes {
				if node.CreatedAt.Before(r.minDate.Time) {
					continue
				}
				hasRecentPosts = true

				r.spin.AddPosts(1)

				reactionNodes := node.Reactions.Nodes
				if node.Reactions.PageInfo.HasPreviousPage {
					moreReactionNodes, err := r.fetchAllReactionsForNode(
						ctx,
						node.ID,
						node.Reactions.PageInfo.StartCursor,
					)
					if err != nil {
						return nil, err
					}
					reactionNodes = append(reactionNodes, moreReactionNodes...)
				}

				objType := ObjectTypeIssueComment
				if node.PullRequest != nil {
					objType = ObjectTypePullRequestComment
				}

				for _, react := range reactionNodes {
					if react.CreatedAt.Before(r.minDate.Time) {
						continue
					}

					reactions = append(reactions, resReaction{
						Reaction: react,
						Parent:   node.NodeMetadata,
						Type:     objType,
					})
					r.spin.AddReactions(1)
				}

				posts, reacts := r.spin.GetCounts()
				r.spin.Progress("fetched %d items and %d reactions", posts, reacts)
			}
		}

		if !hasRecentPosts {
			break
		}

		if issues.PageInfo.HasNextPage && issues.PageInfo.EndCursor != "" {
			args["cursor"] = issues.PageInfo.EndCursor
		} else if issues.PageInfo.HasPreviousPage && issues.PageInfo.StartCursor != "" {
			args["cursor"] = issues.PageInfo.StartCursor
		} else {
			break
		}
	}

	return reactions, nil
}

func (r *requester) fetchRepositoryDiscussionCommentsReactions(
	ctx context.Context,
	repo Repository,
	query string,
) ([]resReaction, error) {
	args := map[string]any{
		"owner":         repo.Owner,
		"name":          repo.Name,
		"last_reaction": 5, // most node has less than 5 reactions
		"nb":            50,
	}

	var reactions []resReaction

	for {
		var res resAPIRepoDiscussionComments
		err := r.client.DoWithContext(ctx, query, args, &res)
		if err != nil {
			return nil, err
		}

		discussions := res.Repository.Discussions
		var hasRecentPosts bool

		for _, discussion := range discussions.Nodes {
			comments := discussion.Comments
			for _, node := range comments.Nodes {
				if node.CreatedAt.Before(r.minDate.Time) {
					continue
				}
				hasRecentPosts = true

				r.spin.AddPosts(1)

				reactionNodes := node.Reactions.Nodes
				if node.Reactions.PageInfo.HasPreviousPage {
					moreReactionNodes, err := r.fetchAllReactionsForNode(
						ctx,
						node.ID,
						node.Reactions.PageInfo.StartCursor,
					)
					if err != nil {
						return nil, err
					}
					reactionNodes = append(reactionNodes, moreReactionNodes...)
				}

				for _, react := range reactionNodes {
					if react.CreatedAt.Before(r.minDate.Time) {
						continue
					}

					reactions = append(reactions, resReaction{
						Reaction: react,
						Parent:   node.NodeMetadata,
						Type:     ObjectTypeDiscussionComment,
					})
					r.spin.AddReactions(1)
				}

				posts, reacts := r.spin.GetCounts()
				r.spin.Progress("fetched %d items and %d reactions", posts, reacts)
			}
		}

		if !hasRecentPosts {
			break
		}

		if discussions.PageInfo.HasNextPage && discussions.PageInfo.EndCursor != "" {
			args["cursor"] = discussions.PageInfo.EndCursor
		} else if discussions.PageInfo.HasPreviousPage && discussions.PageInfo.StartCursor != "" {
			args["cursor"] = discussions.PageInfo.StartCursor
		} else {
			break
		}
	}

	return reactions, nil
}
