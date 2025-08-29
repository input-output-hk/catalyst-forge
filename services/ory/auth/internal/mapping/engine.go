package mapping

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// Engine compiles and evaluates CEL-based mappings for consent and token-hook.
type Engine struct {
	logger     *slog.Logger
	cfg        *FileConfig
	celEnv     *cel.Env
	consent    compiledMapping
	tokenHooks map[string]compiledMapping
	onError    string
	mergeStyle string
}

type compiledMapping struct {
	requirements []cel.Program
	idToken      map[string]cel.Program
	accessExt    map[string]cel.Program
}

// NewEngine loads the YAML config from path and compiles CEL programs.
func NewEngine(logger *slog.Logger, path string) (*Engine, error) {
	fc, err := LoadFileConfig(path)
	if err != nil {
		return nil, err
	}
	e := &Engine{logger: logger, cfg: fc, onError: fc.Policy.OnError, mergeStyle: fc.Policy.MergeStrategy}
	if err := e.buildEnv(); err != nil {
		return nil, err
	}
	cConsent, err := e.compile(fc.Mappings.Consent)
	if err != nil {
		return nil, fmt.Errorf("compile consent mapping: %w", err)
	}
	e.consent = cConsent
	// Compile issuer-scoped token hooks
	e.tokenHooks = map[string]compiledMapping{}
	for iss, m := range fc.Mappings.TokenHooks {
		cm, err := e.compile(m)
		if err != nil {
			return nil, fmt.Errorf("compile token_hooks[%s]: %w", iss, err)
		}
		e.tokenHooks[iss] = cm
	}

	return e, nil
}

func (e *Engine) buildEnv() error {
	// Build CEL environment with dynamic variables and a couple of helpers.
	env, err := cel.NewEnv(
		cel.Variable("kratos", cel.DynType),
		cel.Variable("hydra", cel.DynType),
		cel.Variable("jwt", cel.DynType),
		cel.Variable("req", cel.DynType),
		cel.Function("lower",
			cel.Overload("lower_string", []*cel.Type{cel.StringType}, cel.StringType,
				cel.UnaryBinding(func(v ref.Val) ref.Val {
					s, ok := v.Value().(string)
					if !ok {
						return types.String("")
					}
					return types.String(strings.ToLower(s))
				}))),
	)
	if err != nil {
		return err
	}
	e.celEnv = env
	return nil
}

func (e *Engine) compile(m Mapping) (compiledMapping, error) {
	var out compiledMapping
	// compile requirements
	for _, expr := range m.Requirements {
		ast, iss := e.celEnv.Compile(expr)
		if iss.Err() != nil {
			return out, fmt.Errorf("requirement '%s': %w", expr, iss.Err())
		}
		prg, err := e.celEnv.Program(ast)
		if err != nil {
			return out, err
		}
		out.requirements = append(out.requirements, prg)
	}
	// id_token
	out.idToken = map[string]cel.Program{}
	for k, expr := range m.IDToken {
		prg, err := e.compileOne(expr)
		if err != nil {
			return out, fmt.Errorf("id_token.%s: %w", k, err)
		}
		out.idToken[k] = prg
	}
	// access_token.ext
	out.accessExt = map[string]cel.Program{}
	for k, expr := range m.AccessToken.Ext {
		prg, err := e.compileOne(expr)
		if err != nil {
			return out, fmt.Errorf("access_token.ext.%s: %w", k, err)
		}
		out.accessExt[k] = prg
	}
	return out, nil
}

func (e *Engine) compileOne(expr string) (cel.Program, error) {
	ast, iss := e.celEnv.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	return e.celEnv.Program(ast)
}

// ConsentInput shapes the activation for consent evaluation.
type ConsentInput struct {
	Kratos map[string]any
	Hydra  map[string]any
	Req    map[string]any
}

// TokenHookInput shapes the activation for token hook evaluation.
type TokenHookInput struct {
	JWT map[string]any
	Req map[string]any
}

// EvaluateConsent returns id_token and access_token.ext maps per config.
func (e *Engine) EvaluateConsent(in ConsentInput) (map[string]any, map[string]any, error) {
	act := map[string]any{
		"kratos": in.Kratos,
		"hydra":  in.Hydra,
		"req":    in.Req,
	}
	// requirements
	for _, prg := range e.consent.requirements {
		out, _, err := prg.Eval(act)
		if err != nil {
			return nil, nil, err
		}
		b, ok := out.Value().(bool)
		if !ok || !b {
			return nil, nil, errors.New("consent requirements not satisfied")
		}
	}
	id := map[string]any{}
	for k, prg := range e.consent.idToken {
		v, _, err := prg.Eval(act)
		if err != nil {
			return nil, nil, err
		}
		id[k] = v.Value()
	}
	ext := map[string]any{}
	for k, prg := range e.consent.accessExt {
		v, _, err := prg.Eval(act)
		if err != nil {
			return nil, nil, err
		}
		ext[k] = v.Value()
	}
	return id, ext, nil
}

// EvaluateTokenHookByIssuer selects issuer-scoped mapping and returns access_token.ext.
func (e *Engine) EvaluateTokenHookByIssuer(issuer string, in TokenHookInput) (map[string]any, error) {
	cm, ok := e.tokenHooks[issuer]
	if !ok {
		return nil, fmt.Errorf("unknown issuer: %s", issuer)
	}
	act := map[string]any{
		"jwt": in.JWT,
		"req": in.Req,
	}
	for _, prg := range cm.requirements {
		out, _, err := prg.Eval(act)
		if err != nil {
			return nil, err
		}
		b, ok := out.Value().(bool)
		if !ok || !b {
			return nil, errors.New("token_hook requirements not satisfied")
		}
	}
	ext := map[string]any{}
	for k, prg := range cm.accessExt {
		v, _, err := prg.Eval(act)
		if err != nil {
			return nil, err
		}
		ext[k] = v.Value()
	}
	return ext, nil
}
