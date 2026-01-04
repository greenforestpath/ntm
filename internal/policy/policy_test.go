package policy

import (
	"testing"
)

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()

	blocked, approval, allowed := p.Stats()
	if blocked == 0 {
		t.Error("expected blocked rules in default policy")
	}
	if approval == 0 {
		t.Error("expected approval rules in default policy")
	}
	if allowed == 0 {
		t.Error("expected allowed rules in default policy")
	}
}

func TestCheck_Blocked(t *testing.T) {
	p := DefaultPolicy()

	cases := []struct {
		name    string
		command string
		blocked bool
	}{
		{"git reset --hard", "git reset --hard HEAD", true},
		{"git reset --hard with spaces", "git   reset   --hard", true},
		{"git clean -fd", "git clean -fd", true},
		{"git push --force", "git push --force", true},
		{"git push -f end", "git push origin main -f", true},
		{"git push -f space", "git push -f origin main", true},
		{"rm -rf /", "rm -rf /", true},
		{"rm -rf ~", "rm -rf ~", true},
		{"git branch -D", "git branch -D feature", true},
		{"git stash drop", "git stash drop", true},
		{"git stash clear", "git stash clear", true},

		// Not blocked
		{"git status", "git status", false},
		{"git add", "git add .", false},
		{"git commit", "git commit -m 'test'", false},
		{"git push", "git push origin main", false},
		{"rm file", "rm file.txt", false},
		{"git reset --soft", "git reset --soft HEAD~1", false},
		// force-with-lease is explicitly allowed (takes precedence)
		{"git push --force-with-lease", "git push --force-with-lease", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := p.IsBlocked(tc.command); got != tc.blocked {
				t.Errorf("IsBlocked(%q) = %v, want %v", tc.command, got, tc.blocked)
			}
		})
	}
}

func TestCheck_ApprovalRequired(t *testing.T) {
	p := DefaultPolicy()

	cases := []struct {
		name     string
		command  string
		approval bool
	}{
		{"git rebase -i", "git rebase -i HEAD~3", true},
		{"git commit --amend", "git commit --amend", true},
		{"rm -rf (general)", "rm -rf node_modules", true},

		// Not requiring approval
		{"git status", "git status", false},
		{"git add", "git add .", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := p.NeedsApproval(tc.command); got != tc.approval {
				t.Errorf("NeedsApproval(%q) = %v, want %v", tc.command, got, tc.approval)
			}
		})
	}
}

func TestCheck_Allowed(t *testing.T) {
	p := DefaultPolicy()

	cases := []struct {
		name    string
		command string
		action  Action
	}{
		{"force-with-lease allowed", "git push --force-with-lease origin main", ActionAllow},
		{"soft reset allowed", "git reset --soft HEAD~1", ActionAllow},
		{"mixed reset allowed", "git reset HEAD~1", ActionAllow},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			match := p.Check(tc.command)
			if match == nil {
				t.Errorf("Check(%q) = nil, want Action=%v", tc.command, tc.action)
				return
			}
			if match.Action != tc.action {
				t.Errorf("Check(%q).Action = %v, want %v", tc.command, match.Action, tc.action)
			}
		})
	}
}

func TestCheck_Precedence(t *testing.T) {
	// Allowed should take precedence over blocked
	p := DefaultPolicy()

	// git push --force-with-lease contains --force which could match block pattern
	// but should be allowed due to explicit allow rule
	match := p.Check("git push --force-with-lease")
	if match == nil {
		t.Error("expected match for git push --force-with-lease")
		return
	}
	if match.Action != ActionAllow {
		t.Errorf("expected ActionAllow, got %v", match.Action)
	}
}

func TestCheck_NoMatch(t *testing.T) {
	p := DefaultPolicy()

	match := p.Check("ls -la")
	if match != nil {
		t.Errorf("Check('ls -la') = %v, want nil", match)
	}
}
