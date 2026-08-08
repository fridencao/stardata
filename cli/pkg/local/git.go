package local

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/fridencao/stardata/cli/pkg/cmdutil"
	localv1 "github.com/fridencao/stardata/proto/gen/stardata/local/v1"
	"github.com/fridencao/stardata/runtime/pkg/gitutil"
)

func (s *Server) GitStatus(ctx context.Context, r *connect.Request[localv1.GitStatusRequest]) (*connect.Response[localv1.GitStatusResponse], error) {
	gitPath, subPath, err := gitutil.InferRepoRootAndSubpath(s.app.ProjectPath)
	if err != nil {
		// Not a git repo
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	// Get authenticated admin client
	if !s.app.ch.IsAuthenticated() {
		// if not authenticated do not return local/remote changes info
		st, err := gitutil.Status(ctx, gitPath, subPath, "origin", "")
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&localv1.GitStatusResponse{
			Branch:     st.Branch,
			GithubUrl:  st.RemoteURL,
			Subpath:    subPath,
			ManagedGit: false,
		}), nil
	}

	// TODO: cache project inference
	projects, err := s.app.ch.InferProjects(ctx, s.app.ch.Org, s.app.ProjectPath)
	if err != nil {
		if !errors.Is(err, cmdutil.ErrInferProjectFailed) {
			return nil, err
		}
		// if not connected to a project do not return local/remote changes info
		st, err := gitutil.Status(ctx, gitPath, subPath, "origin", "")
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&localv1.GitStatusResponse{
			Branch:     st.Branch,
			GithubUrl:  st.RemoteURL,
			Subpath:    subPath,
			ManagedGit: false,
		}), nil
	}
	project := projects[0]

	if subPath != project.Subpath {
		// unlikely but just in case
		return nil, connect.NewError(connect.CodeUnknown, errors.New("detected subpath within git repo does not match project subpath"))
	}

	// get ephemeral git credentials
	config, err := s.app.ch.GitHelper(s.app.ch.Org, project.Name, gitPath).GitConfig(ctx)
	if err != nil {
		return nil, err
	}
	// set remote
	// usually not needed but the older flow did not set the remote by name `rill`
	err = gitutil.SetRemote(gitPath, config)
	if err != nil {
		return nil, err
	}
	err = gitutil.Fetch(ctx, gitPath, config)
	if err != nil {
		return nil, err
	}
	gs, err := gitutil.Status(ctx, gitPath, subPath, config.RemoteName(), "")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&localv1.GitStatusResponse{
		Branch:        gs.Branch,
		GithubUrl:     gs.RemoteURL,
		Subpath:       subPath,
		ManagedGit:    config.ManagedRepo,
		LocalChanges:  gs.LocalChanges,
		LocalCommits:  gs.LocalCommits,
		RemoteCommits: gs.RemoteCommits,
	}), nil
}

func (s *Server) GitPull(ctx context.Context, r *connect.Request[localv1.GitPullRequest]) (*connect.Response[localv1.GitPullResponse], error) {
	// Get authenticated admin client
	if !s.app.ch.IsAuthenticated() {
		return nil, errors.New("must authenticate before performing this action")
	}

	projects, err := s.app.ch.InferProjects(ctx, s.app.ch.Org, s.app.ProjectPath)
	if err != nil {
		if !errors.Is(err, cmdutil.ErrInferProjectFailed) {
			return nil, err
		}
		return nil, errors.New("repo is not connected to a project")
	}
	project := projects[0]

	gitPath, subpath, err := gitutil.InferRepoRootAndSubpath(s.app.ProjectPath)
	if err != nil {
		// Not a git repo
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if project.Subpath != subpath {
		return nil, errors.New("detected subpath within git repo does not match project subpath")
	}

	config, err := s.app.ch.GitHelper(s.app.ch.Org, project.Name, gitPath).GitConfig(ctx)
	if err != nil {
		return nil, err
	}
	err = gitutil.SetRemote(gitPath, config)
	if err != nil {
		return nil, err
	}

	remote, err := config.FullyQualifiedRemote()
	if err != nil {
		return nil, err
	}

	out, err := gitutil.Pull(ctx, gitPath, r.Msg.DiscardLocal, remote, config.RemoteName())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&localv1.GitPullResponse{
		Output: out,
	}), nil
}

func (s *Server) GitPush(ctx context.Context, r *connect.Request[localv1.GitPushRequest]) (*connect.Response[localv1.GitPushResponse], error) {
	// Get authenticated admin client
	if !s.app.ch.IsAuthenticated() {
		return nil, errors.New("must authenticate before performing this action")
	}

	projects, err := s.app.ch.InferProjects(ctx, s.app.ch.Org, s.app.ProjectPath)
	if err != nil {
		if !errors.Is(err, cmdutil.ErrInferProjectFailed) {
			return nil, err
		}
		return nil, errors.New("repo is not connected to a project")
	}
	project := projects[0]

	gitPath, subpath, err := gitutil.InferRepoRootAndSubpath(s.app.ProjectPath)
	// Not a git repo
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if project.Subpath != subpath {
		return nil, errors.New("detected subpath within git repo does not match project subpath")
	}

	author, err := s.app.ch.GitSignature(ctx, gitPath)
	if err != nil {
		return nil, err
	}

	config, err := s.app.ch.GitHelper(s.app.ch.Org, project.Name, gitPath).GitConfig(ctx)
	if err != nil {
		return nil, err
	}
	err = gitutil.SetRemote(gitPath, config)
	if err != nil {
		return nil, err
	}

	// fetch the status again
	gs, err := gitutil.Status(ctx, gitPath, subpath, config.RemoteName(), "")
	if err != nil {
		return nil, err
	}
	if gs.RemoteCommits > 0 && !r.Msg.Force {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot push with remote commits present, please pull first"))
	}

	var choice string
	if r.Msg.Force {
		choice = "2"
	} else {
		choice = "1"
	}
	err = s.app.ch.CommitAndSafePush(ctx, gitPath, config, r.Msg.CommitMessage, author, choice)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&localv1.GitPushResponse{}), nil
}
