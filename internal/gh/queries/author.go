package queries

import _ "embed"

var (
	//go:embed author_pull_requests.graphql
	AuthorPullRequests string

	//go:embed author_issues.graphql
	AuthorIssues string

	//go:embed author_issue_comments.graphql
	AuthorIssueComments string

	//go:embed author_repository_discussions.graphql
	AuthorRepositoryDiscussions string

	//go:embed author_repository_discussion_comments.graphql
	AuthorRepositoryDiscussionComments string

	//go:embed author_commit_comments.graphql
	AuthorCommitComments string

	//go:embed author_reactions_by_node.graphql
	AuthorReactionsByNode string
)
