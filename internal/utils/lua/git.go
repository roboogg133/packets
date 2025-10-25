package utils_lua

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	lua "github.com/yuin/gopher-lua"
)

func LGitClone(L *lua.LState) int {
	uri := L.CheckString(1)
	output := L.CheckString(2)

	depth := 1
	if L.GetTop() > 2 {
		depth = L.CheckInt(3)
	}

	_, err := git.PlainClone(output, &git.CloneOptions{
		URL:      uri,
		Progress: os.Stdout,
		Depth:    depth,
	})
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func LGitCheckout(L *lua.LState) int {
	dir := L.CheckString(1)
	branchorid := L.CheckString(2)

	repo, err := git.PlainOpen(dir)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	worktree, err := repo.Worktree()
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	if err := tryCheckout(*worktree, branchorid); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}
func LGitPUll(L *lua.LState) int {
	dir := L.CheckString(1)

	depth := 1
	if L.GetTop() > 1 {
		depth = L.CheckInt(2)
	}

	repo, err := git.PlainOpen(dir)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	worktree, err := repo.Worktree()
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	if err := worktree.Pull(&git.PullOptions{Depth: depth}); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func tryCheckout(worktree git.Worktree, reference string) error {
	err := worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(reference),
	})
	if err == nil {
		return nil
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + reference),
	})
	if err == nil {
		return nil
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/tags/" + reference),
	})
	if err == nil {
		return nil
	}

	hash := plumbing.NewHash(reference)
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err == nil {
		return nil
	}

	return fmt.Errorf("cannot checkout '%s' as branch, tag, or commit", reference)
}
