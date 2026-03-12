package db

import (
	"fmt"
	"testing"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()

	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	if err := database.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	return database
}

func createAgentForTests(t *testing.T, database *DB, id, apiKey string) {
	t.Helper()
	if err := database.CreateAgent(id, apiKey); err != nil {
		t.Fatalf("CreateAgent(%q) error = %v", id, err)
	}
}

func TestOpenAndMigrate(t *testing.T) {
	database := newTestDB(t)
	if database == nil {
		t.Fatal("expected database instance")
	}
}

func TestCreateAgentAndGetAgentByAPIKey(t *testing.T) {
	database := newTestDB(t)
	createAgentForTests(t, database, "agent-a", "key-a")

	agent, err := database.GetAgentByAPIKey("key-a")
	if err != nil {
		t.Fatalf("GetAgentByAPIKey() error = %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent, got nil")
	}
	if agent.ID != "agent-a" {
		t.Fatalf("agent.ID = %q, want %q", agent.ID, "agent-a")
	}
	if agent.APIKey != "key-a" {
		t.Fatalf("agent.APIKey = %q, want %q", agent.APIKey, "key-a")
	}
}

func TestGetAgentByAPIKeyMissingReturnsNil(t *testing.T) {
	database := newTestDB(t)

	agent, err := database.GetAgentByAPIKey("missing")
	if err != nil {
		t.Fatalf("GetAgentByAPIKey() error = %v", err)
	}
	if agent != nil {
		t.Fatalf("expected nil agent, got %+v", *agent)
	}
}

func TestInsertCommitAndGetCommit(t *testing.T) {
	database := newTestDB(t)
	createAgentForTests(t, database, "agent-a", "key-a")

	if err := database.InsertCommit("a1", "", "agent-a", "root"); err != nil {
		t.Fatalf("InsertCommit(root) error = %v", err)
	}
	if err := database.InsertCommit("b1", "a1", "agent-a", "child"); err != nil {
		t.Fatalf("InsertCommit(child) error = %v", err)
	}

	commit, err := database.GetCommit("b1")
	if err != nil {
		t.Fatalf("GetCommit() error = %v", err)
	}
	if commit == nil {
		t.Fatal("expected commit, got nil")
	}
	if commit.ParentHash != "a1" {
		t.Fatalf("commit.ParentHash = %q, want %q", commit.ParentHash, "a1")
	}
}

func TestListCommitsReturnsAllWhenAgentFilterEmpty(t *testing.T) {
	database := newTestDB(t)
	createAgentForTests(t, database, "agent-a", "key-a")
	createAgentForTests(t, database, "agent-b", "key-b")

	if err := database.InsertCommit("c1", "", "agent-a", "a"); err != nil {
		t.Fatalf("InsertCommit(c1) error = %v", err)
	}
	if err := database.InsertCommit("c2", "", "agent-b", "b"); err != nil {
		t.Fatalf("InsertCommit(c2) error = %v", err)
	}

	commits, err := database.ListCommits("", 0, 0)
	if err != nil {
		t.Fatalf("ListCommits() error = %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("len(commits) = %d, want %d", len(commits), 2)
	}
}

func TestListCommitsFiltersByAgent(t *testing.T) {
	database := newTestDB(t)
	createAgentForTests(t, database, "agent-a", "key-a")
	createAgentForTests(t, database, "agent-b", "key-b")

	if err := database.InsertCommit("c1", "", "agent-a", "a"); err != nil {
		t.Fatalf("InsertCommit(c1) error = %v", err)
	}
	if err := database.InsertCommit("c2", "", "agent-b", "b"); err != nil {
		t.Fatalf("InsertCommit(c2) error = %v", err)
	}

	commits, err := database.ListCommits("agent-a", 10, 0)
	if err != nil {
		t.Fatalf("ListCommits() error = %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("len(commits) = %d, want %d", len(commits), 1)
	}
	if commits[0].AgentID != "agent-a" {
		t.Fatalf("commits[0].AgentID = %q, want %q", commits[0].AgentID, "agent-a")
	}
}

func TestGetChildrenReturnsDirectChildren(t *testing.T) {
	database := newTestDB(t)
	createAgentForTests(t, database, "agent-a", "key-a")

	if err := database.InsertCommit("root", "", "agent-a", "root"); err != nil {
		t.Fatalf("InsertCommit(root) error = %v", err)
	}
	if err := database.InsertCommit("child-1", "root", "agent-a", "child-1"); err != nil {
		t.Fatalf("InsertCommit(child-1) error = %v", err)
	}
	if err := database.InsertCommit("child-2", "root", "agent-a", "child-2"); err != nil {
		t.Fatalf("InsertCommit(child-2) error = %v", err)
	}

	children, err := database.GetChildren("root")
	if err != nil {
		t.Fatalf("GetChildren() error = %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("len(children) = %d, want %d", len(children), 2)
	}
}

func TestGetChildrenMissingReturnsEmptySlice(t *testing.T) {
	database := newTestDB(t)

	children, err := database.GetChildren("does-not-exist")
	if err != nil {
		t.Fatalf("GetChildren() error = %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("len(children) = %d, want %d", len(children), 0)
	}
}

func TestGetLineageReturnsCommitChain(t *testing.T) {
	database := newTestDB(t)
	createAgentForTests(t, database, "agent-a", "key-a")

	if err := database.InsertCommit("root", "", "agent-a", "root"); err != nil {
		t.Fatalf("InsertCommit(root) error = %v", err)
	}
	if err := database.InsertCommit("mid", "root", "agent-a", "mid"); err != nil {
		t.Fatalf("InsertCommit(mid) error = %v", err)
	}
	if err := database.InsertCommit("leaf", "mid", "agent-a", "leaf"); err != nil {
		t.Fatalf("InsertCommit(leaf) error = %v", err)
	}

	lineage, err := database.GetLineage("leaf")
	if err != nil {
		t.Fatalf("GetLineage() error = %v", err)
	}
	if len(lineage) != 3 {
		t.Fatalf("len(lineage) = %d, want %d", len(lineage), 3)
	}
	if lineage[0].Hash != "leaf" || lineage[1].Hash != "mid" || lineage[2].Hash != "root" {
		t.Fatalf("unexpected lineage order: %+v", lineage)
	}
}

func TestGetLineageMissingReturnsEmpty(t *testing.T) {
	database := newTestDB(t)

	lineage, err := database.GetLineage("missing")
	if err != nil {
		t.Fatalf("GetLineage() error = %v", err)
	}
	if len(lineage) != 0 {
		t.Fatalf("len(lineage) = %d, want %d", len(lineage), 0)
	}
}

func TestCreateChannelAndListChannels(t *testing.T) {
	database := newTestDB(t)

	if err := database.CreateChannel("general", "general channel"); err != nil {
		t.Fatalf("CreateChannel(general) error = %v", err)
	}
	if err := database.CreateChannel("dev", "dev channel"); err != nil {
		t.Fatalf("CreateChannel(dev) error = %v", err)
	}

	channels, err := database.ListChannels()
	if err != nil {
		t.Fatalf("ListChannels() error = %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("len(channels) = %d, want %d", len(channels), 2)
	}
	if channels[0].Name != "dev" || channels[1].Name != "general" {
		t.Fatalf("unexpected channel order: %+v", channels)
	}
}

func TestCreatePostAndListPosts(t *testing.T) {
	database := newTestDB(t)
	createAgentForTests(t, database, "agent-a", "key-a")

	if err := database.CreateChannel("general", ""); err != nil {
		t.Fatalf("CreateChannel() error = %v", err)
	}
	ch, err := database.GetChannelByName("general")
	if err != nil || ch == nil {
		t.Fatalf("GetChannelByName() = (%v, %v), want non-nil channel", ch, err)
	}

	post, err := database.CreatePost(ch.ID, "agent-a", nil, "hello")
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	if post == nil {
		t.Fatal("expected post, got nil")
	}

	posts, err := database.ListPosts(ch.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListPosts() error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want %d", len(posts), 1)
	}
	if posts[0].Content != "hello" {
		t.Fatalf("posts[0].Content = %q, want %q", posts[0].Content, "hello")
	}
}

func TestListPostsDefaultLimitApplies(t *testing.T) {
	database := newTestDB(t)
	createAgentForTests(t, database, "agent-a", "key-a")

	if err := database.CreateChannel("general", ""); err != nil {
		t.Fatalf("CreateChannel() error = %v", err)
	}
	ch, err := database.GetChannelByName("general")
	if err != nil || ch == nil {
		t.Fatalf("GetChannelByName() = (%v, %v), want non-nil channel", ch, err)
	}

	for i := 0; i < 55; i++ {
		if _, err := database.CreatePost(ch.ID, "agent-a", nil, fmt.Sprintf("post-%d", i)); err != nil {
			t.Fatalf("CreatePost(%d) error = %v", i, err)
		}
	}

	posts, err := database.ListPosts(ch.ID, 0, 0)
	if err != nil {
		t.Fatalf("ListPosts() error = %v", err)
	}
	if len(posts) != 50 {
		t.Fatalf("len(posts) = %d, want %d", len(posts), 50)
	}
}

func TestGetRepliesReturnsOnlyReplyPosts(t *testing.T) {
	database := newTestDB(t)
	createAgentForTests(t, database, "agent-a", "key-a")

	if err := database.CreateChannel("general", ""); err != nil {
		t.Fatalf("CreateChannel() error = %v", err)
	}
	ch, err := database.GetChannelByName("general")
	if err != nil || ch == nil {
		t.Fatalf("GetChannelByName() = (%v, %v), want non-nil channel", ch, err)
	}

	root, err := database.CreatePost(ch.ID, "agent-a", nil, "root")
	if err != nil {
		t.Fatalf("CreatePost(root) error = %v", err)
	}

	if _, err := database.CreatePost(ch.ID, "agent-a", &root.ID, "reply-1"); err != nil {
		t.Fatalf("CreatePost(reply-1) error = %v", err)
	}
	if _, err := database.CreatePost(ch.ID, "agent-a", &root.ID, "reply-2"); err != nil {
		t.Fatalf("CreatePost(reply-2) error = %v", err)
	}
	if _, err := database.CreatePost(ch.ID, "agent-a", nil, "sibling-root"); err != nil {
		t.Fatalf("CreatePost(sibling-root) error = %v", err)
	}

	replies, err := database.GetReplies(root.ID)
	if err != nil {
		t.Fatalf("GetReplies() error = %v", err)
	}
	if len(replies) != 2 {
		t.Fatalf("len(replies) = %d, want %d", len(replies), 2)
	}
	for i, reply := range replies {
		if reply.ParentID == nil || *reply.ParentID != root.ID {
			t.Fatalf("replies[%d].ParentID = %v, want %d", i, reply.ParentID, root.ID)
		}
	}
}
