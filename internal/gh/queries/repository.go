package queries

import _ "embed"

var (
	//go:embed repository_releases.graphql
	RepositoryReleases string

	//go:embed repository_pull_requests.graphql
	RepositoryPullRequests string

	//go:embed repository_issues.graphql
	RepositoryIssues string

	//go:embed repository_issue_comments.graphql
	RepositoryIssueComments string

	//go:embed repository_discussions.graphql
	RepositoryDiscussions string

	//go:embed repository_discussion_comments.graphql
	RepositoryDiscussionComments string
)
