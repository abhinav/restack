package restack

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			Desc:           "issue 41", // https://github.com/abhinav/restack/issues/41
			RemoteName:     "origin",
			RebaseHeadName: "stack",
			Branches: []Branch{
				{Name: "5601-connection-refused", Hash: "29b83a30c"},
				{Name: "5460-publish-bundle", Hash: "ae23c4203"},
				{Name: "5460-publish-bundle-client", Hash: "ea9b3946b"},
			},
			Give: []string{
				"pick eaed5a16a <Kris Kowal> fix(agoric-cli): Thread rpcAddresses for Cosmos publishBundle",
				"pick bd49c28ed <Kris Kowal> fix(agoric-cli): Follow-up: thread random as power",
				"pick da0626a7e <Kris Kowal> fix(agoric-cli): Follow-up: conditionally coerce RPC addresses",
				"pick 29b83a30c <Kris Kowal> fix(agoric-cli): Follow-up: heuristic for distinguishing bare hostnames from URLs (5601-connection-refused)",
				"pick e2b9551ba <Kris Kowal> fix(cosmic-swingset): Publish installation success and failure topic",
				"pick d8685a017 <Kris Kowal> fix(cosmic-swingset): Follow-up: use new pubsub mechanism",
				"pick a8ca6046f <Kris Kowal> fix(cosmic-swingset): Follow-up comment from Richard Gibson",
				"pick 33445b5b7 <Kris Kowal> fix(cosmic-swingset): Follow-up: Publish base value to installation topic",
				"pick 021edf49e <Kris Kowal> fix(cosmic-swingset): Follow-up: prettier",
				"pick d0af526c4 <Kris Kowal> fix(cosmic-swingset): Follow-up: should publish sequences",
				"pick ae23c4203 <Kris Kowal> refactor: Thread solo home directory more generally (5460-publish-bundle)",
				"pick c8090b4ab <Kris Kowal> refactor(agoric-cli): Publish with RPC instead of agd subshell",
				"pick 8cd36f76e <Kris Kowal> fix(casting): iterateLatest erroneously adapted getEachIterable",
				"pick fe113b6bc <Kris Kowal> fix(casting): Release all I/O handles between yield and next",
				"pick 00d84a3e1 <Kris Kowal> feat(agoric-cli): Reveal block heights to agoric follow, opt-in for lossy",
				"pick 2c04f46aa <Kris Kowal> chore: Update yarn.lock",
				"pick ea9b3946b <Kris Kowal> chore: yarn deduplicate (HEAD -> stack, 5460-publish-bundle-client)",
				"",
				"# Rebase comment starts here",
			},
			Want: []string{
				"pick eaed5a16a <Kris Kowal> fix(agoric-cli): Thread rpcAddresses for Cosmos publishBundle",
				"pick bd49c28ed <Kris Kowal> fix(agoric-cli): Follow-up: thread random as power",
				"pick da0626a7e <Kris Kowal> fix(agoric-cli): Follow-up: conditionally coerce RPC addresses",
				"pick 29b83a30c <Kris Kowal> fix(agoric-cli): Follow-up: heuristic for distinguishing bare hostnames from URLs (5601-connection-refused)",
				"exec git branch -f 5601-connection-refused",
				"",
				"pick e2b9551ba <Kris Kowal> fix(cosmic-swingset): Publish installation success and failure topic",
				"pick d8685a017 <Kris Kowal> fix(cosmic-swingset): Follow-up: use new pubsub mechanism",
				"pick a8ca6046f <Kris Kowal> fix(cosmic-swingset): Follow-up comment from Richard Gibson",
				"pick 33445b5b7 <Kris Kowal> fix(cosmic-swingset): Follow-up: Publish base value to installation topic",
				"pick 021edf49e <Kris Kowal> fix(cosmic-swingset): Follow-up: prettier",
				"pick d0af526c4 <Kris Kowal> fix(cosmic-swingset): Follow-up: should publish sequences",
				"pick ae23c4203 <Kris Kowal> refactor: Thread solo home directory more generally (5460-publish-bundle)",
				"exec git branch -f 5460-publish-bundle",
				"",
				"pick c8090b4ab <Kris Kowal> refactor(agoric-cli): Publish with RPC instead of agd subshell",
				"pick 8cd36f76e <Kris Kowal> fix(casting): iterateLatest erroneously adapted getEachIterable",
				"pick fe113b6bc <Kris Kowal> fix(casting): Release all I/O handles between yield and next",
				"pick 00d84a3e1 <Kris Kowal> feat(agoric-cli): Reveal block heights to agoric follow, opt-in for lossy",
				"pick 2c04f46aa <Kris Kowal> chore: Update yarn.lock",
				"pick ea9b3946b <Kris Kowal> chore: yarn deduplicate (HEAD -> stack, 5460-publish-bundle-client)",
				"exec git branch -f 5460-publish-bundle-client",
				"",
				"# Uncomment this section to push the changes.",
				"# exec git push -f origin 5601-connection-refused",
				"# exec git push -f origin 5460-publish-bundle",
				"# exec git push -f origin 5460-publish-bundle-client",
				"",
				"# Rebase comment starts here",
			},
		},
		{
			Desc:           "comment after instructions",
			RemoteName:     "origin",
			RebaseHeadName: "feature/wip",
			Branches: []Branch{
				{Name: "feature/1", Hash: "hash1"},
				{Name: "feature/2", Hash: "hash2"},
			},
			Give: []string{
				"pick hash1 Implement feature 1",
				"pick hash2 Implement feature 2",
				"# Rebase instructions",
			},
			Want: []string{
				"pick hash1 Implement feature 1",
				"exec git branch -f feature/1",
				"",
				"pick hash2 Implement feature 2",
				"exec git branch -f feature/2",
				"",
				"# Uncomment this section to push the changes.",
				"# exec git push -f origin feature/1",
				"# exec git push -f origin feature/2",
				"",
				"# Rebase instructions",
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
			require.NoError(t,
				r.Restack(ctx, &Request{
					RemoteName: tt.RemoteName,
					From:       src,
					To:         &dst,
				}),
				"restacker failed")

			want := append(tt.Want, "")
			got := strings.Split(dst.String(), "\n")
			assert.Equal(t, want, got, "output mismatch")
		})
	}
}
