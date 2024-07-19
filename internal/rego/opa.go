// Copyright (C) 2022 AlgoNode Org.
//
// algostreamer is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// algostreamer is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with algostreamer.  If not, see <https://www.gnu.org/licenses/>.
package rego

import (
	"fmt"
	"os"

	"github.com/open-policy-agent/opa/ast"
)

type RegoRulesMap struct {
	Status string `json:"status"`
	Block  string `json:"block"`
	Tx     string `json:"tx"`
}

type RegoRulesCompilers struct {
	Status *ast.Compiler
	Block  *ast.Compiler
	Tx     *ast.Compiler
}

type OpaConfig struct {
	MyID  string       `json:"myid"`
	Rules RegoRulesMap `json:"rules"`
	c     RegoRulesCompilers
}

func CompileCfg(cfg *OpaConfig) error {
	if err := compileRegoFile(cfg.Rules.Status, "status", &cfg.c.Status); err != nil {
		return err
	}
	if err := compileRegoFile(cfg.Rules.Block, "block", &cfg.c.Block); err != nil {
		return err
	}
	if err := compileRegoFile(cfg.Rules.Tx, "tx", &cfg.c.Tx); err != nil {
		return err
	}
	if cfg.c.Status == nil && cfg.c.Block == nil && cfg.c.Tx == nil {
		return fmt.Errorf("define OPA rule file for at least one event category (status|block|tx)")
	}
	return nil
}

func compileRegoFile(file string, module string, compiler **ast.Compiler) error {
	if file == "" {
		return nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	c, err := ast.CompileModules(map[string]string{
		module: string(data),
	})

	if err != nil {
		return err
	}
	fmt.Printf("File %s compiled\n", file)
	*compiler = c
	return nil

}
