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

func TestRestacker(t *testing.T) {
	tests := []struct {
		Desc           string
		RemoteName     string
		RebaseHeadName string
		KnownHeads     map[string][]string

		Give []string
		Want []string
	}{
		{
			Desc:           "No matches",
			RemoteName:     "origin",
			RebaseHeadName: "feature",
			KnownHeads: map[string][]string{
				"hash1": []string{"feature"},
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
			KnownHeads: map[string][]string{
				"hash1": []string{"feature/1", "feature/wip"},
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
			KnownHeads: map[string][]string{
				"hash1": []string{"feature/1"},
				"hash3": []string{"feature/2"},
				"hash7": []string{"feature/3"},
				"hash9": []string{"feature/wip"},
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
				ListHeads(gomock.Any()).
				Return(tt.KnownHeads, nil)

			src := bytes.NewBufferString(strings.Join(tt.Give, "\n") + "\n")
			var dst bytes.Buffer

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			r := Restacker{RemoteName: tt.RemoteName, Git: mockGit}
			err := r.Run(ctx, &dst, src)
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
