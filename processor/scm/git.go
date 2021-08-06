package scm

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/pkg/errors"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/log"
	"go.uber.org/zap"
	"io"
)

type GitScm struct {
}

func (s *GitScm) getAuth(cfg *v1.ScmCfg) (auth transport.AuthMethod, err error) {
	auth = nil

	if cfg.Credit == nil {
		return
	}

	switch cfg.Credit.Type {
	case v1.CreditType_NoCredit:
	case v1.CreditType_TypeUserPwd:
		auth = &http.BasicAuth{
			Username: cfg.Credit.Username,
			Password: cfg.Credit.Password,
		}
	case v1.CreditType_TypeSSHPrivateKey:
		auth, err = ssh.NewPublicKeys("git", []byte(cfg.Credit.PrivateKey), cfg.Credit.Password)
	default:
		auth = &http.TokenAuth{Token: cfg.Credit.Password}
	}

	return
}

func (s *GitScm) Clone(ctx context.Context, workDir string, cfg *v1.ScmCfg, logWriter io.Writer) error {
	dir := workDir

	auth, err := s.getAuth(cfg)

	if err != nil {
		return errors.Wrapf(err, "解析代码拉取凭证失败")
	}

	repo, err := git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
		URL:               cfg.Address,
		Auth:              auth,
		ReferenceName:     plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", cfg.Branch)),
		SingleBranch:      true,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          logWriter,
		InsecureSkipTLS:   true,
	})

	log.GetLogger().Info("git clone result", zap.Any("repo", repo), zap.Error(err))

	if err != nil {
		return errors.Wrapf(err, "代码拉取失败")
	}

	return nil
}
