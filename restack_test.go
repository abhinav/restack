package restack

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func TestGitRestacker(t *testing.T) {
	tests := []struct {
		Desc           string
		RemoteName     string
		RebaseHeadName string
		Branches       []Branch

		Give []string
		Want []string
	}{
		{
			Desc:           "No matches",
			RemoteName:     "origin",
			RebaseHeadName: "feature",
			Branches: []Branch{
				{Name: "feature", Hash: "hash1"},
			},
			Give: []string{
				"pick hash0 Do something",
				"exec make test",
				"pick hash1 Implement feature",
			},
			Want: []string{
				"pick hash0 Do something",
				"exec make test",
				"pick hash1 Implement feature",
			},
		},
		{
			Desc: "Bad pick instruction",
			Give: []string{"pick"},
			Want: []string{"pick"},
		},
		{
			Desc:           "Branch at rebase head",
			RemoteName:     "origin",
			RebaseHeadName: "feature/wip",
			Branches: []Branch{
				{Name: "feature/1", Hash: "hash1"},
				{Name: "feature/wip", Hash: "hash1"},
			},
			Give: []string{
				"pick hash0 Do something",
				"exec make test",
				"pick hash1 Implement feature",
				"exec make test",
				"",
				"# Rebase instructions",
			},
			Want: []string{
				"pick hash0 Do something",
				"exec make test",
				"pick hash1 Implement feature",
				"exec git branch -f feature/1",
				"",
				"exec make test",
				"",
				"# Uncomment this section to push the changes.",
				"# exec git push -f origin feature/1",
				"",
				"# Rebase instructions",
			},
		},
		{
			Desc:           "Rebase instructions missing",
			RemoteName:     "origin",
			RebaseHeadName: "feature/wip",
			Branches: []Branch{
				{Name: "feature/1", Hash: "hash1"},
				{Name: "feature/2", Hash: "hash3"},
				{Name: "feature/3", Hash: "hash7"},
				{Name: "feature/wip", Hash: "hash9"},
			},
			Give: []string{
				"pick hash0 Do something 0",
				"pick hash1 Implement feature1",
				"pick hash2 Do something",
				"pick hash3 Implement feature2",
				"pick hash4 Do something 4",
				"pick hash5 Do something 5",
				"pick hash6 Do something 6",
				"pick hash7 Implement feature3",
				"pick hash8 Do something 8",
				"pick hash9 Do something 9",
			},
			Want: []string{
				"pick hash0 Do something 0",
				"pick hash1 Implement feature1",
				"exec git branch -f feature/1",
				"",
				"pick hash2 Do something",
				"pick hash3 Implement feature2",
				"exec git branch -f feature/2",
				"",
				"pick hash4 Do something 4",
				"pick hash5 Do something 5",
				"pick hash6 Do something 6",
				"pick hash7 Implement feature3",
				"exec git branch -f feature/3",
				"",
				"pick hash8 Do something 8",
				"pick hash9 Do something 9",
				"",
				"# Uncomment this section to push the changes.",
				"# exec git push -f origin feature/1",
				"# exec git push -f origin feature/2",
				"# exec git push -f origin feature/3",
			},
		},
		{
			Desc:           "fixup commit",
			RebaseHeadName: "b",
			Branches: []Branch{
				{Name: "a", Hash: "hash1"},
				{Name: "b", Hash: "hash3"},
			},
			Give: []string{
				"pick hash0 do thing",
				"pick hash1 another thing",
				"fixup hash2 stuff",
				"pick hash3 whatever",
			},
			Want: []string{
				"pick hash0 do thing",
				"pick hash1 another thing",
				"fixup hash2 stuff",
				"exec git branch -f a",
				"",
				"pick hash3 whatever",
			},
		},
		{
			Desc:           "squash commit",
			RebaseHeadName: "b",
			Branches: []Branch{
				{Name: "a", Hash: "hash1"},
				{Name: "b", Hash: "hash3"},
			},
			Give: []string{
				"pick hash0 do thing",
				"pick hash1 another thing",
				"squash hash2 stuff",
				"pick hash3 whatever",
			},
			Want: []string{
				"pick hash0 do thing",
				"pick hash1 another thing",
				"squash hash2 stuff",
				"exec git branch -f a",
				"",
				"pick hash3 whatever",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockGit := NewMockGit(mockCtrl)
			mockGit.EXPECT().
				RebaseHeadName(gomock.Any()).
				Return(tt.RebaseHeadName, nil)
			mockGit.EXPECT().
				ListBranches(gomock.Any()).
				Return(tt.Branches, nil)

			src := bytes.NewBufferString(strings.Join(tt.Give, "\n") + "\n")
			var dst bytes.Buffer

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			r := GitRestacker{Git: mockGit}
			err := r.Restack(ctx, &Request{
				RemoteName: tt.RemoteName,
				From:       src,
				To:         &dst,
			})
			if err != nil {
				t.Fatalf("restacker failed: %v", err)
			}

			want := append(tt.Want, "")
			got := strings.Split(dst.String(), "\n")
			if diff := cmp.Diff(want, got); len(diff) > 0 {
				t.Errorf("output: (-want, +got):\n%s", diff)
			}
		})
	}
}
