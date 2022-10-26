// Copyright 2011 Bobby Powers. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// based off of Appendix A from http://dinosaur.compilertools.net/yacc/

%{
/*
Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package geminiql

import (
	"strconv"
)

func updateStmt(QLlex interface{}, stmt Statement) {
    QLlex.(*QLLexerImpl).UpdateStmt(stmt)
}

%}

// fields inside this union end up as the fields in a structure known
// as ${PREFIX}SymType, of which a reference is passed to the lexer.
%union{
    stmts []Statement
    stmt Statement
    str string
    strslice []string
    integer int64
    decimal float64
    pair Pair
    pairs Pairs
}

// any non-terminal which returns a value needs a type, which is
// really a field name in the above union struct
%type <stmts> STATEMENTS
%type <stmt> INSERT_STATEMENT USE_STATEMENT SET_STATEMENT
%type <str> LINE_PROTOCOL TIME_SERIE MEASUREMENT KV_RAW KV_RAWS TIME
%type <strslice> NAMESPACE
%type <pair> KEY_VALUE
%type <pairs> KEY_VALUES

// same for terminals
%token <str> INSERT INTO USE SET
%token <str> DOT COMMA
%token <str> EQ
%token <str> IDENT
%token <integer> INTEGER
%token <decimal> DECIMAL
%token <str> STRING
%token <str> RAW

%%

STATEMENTS:
    INSERT_STATEMENT
    {
        updateStmt(QLlex, $1)
    }
    |USE_STATEMENT
    {
        updateStmt(QLlex, $1)
    }
    |SET_STATEMENT
    {
        updateStmt(QLlex, $1)
    }

SET_STATEMENT:
    SET KEY_VALUES
    {
        stmt := &SetStatement{}
        stmt.KVS = $2
        $$ = stmt
    }

USE_STATEMENT:
    USE NAMESPACE
    {
        stmt := &UseStatement{}
        if len($2) == 1 {
            stmt.DB = $2[0]
            $$ = stmt
        } else if len($2) == 2 {
            stmt.DB = $2[0]
            stmt.RP = $2[1]
            $$ = stmt
        } else {
            QLlex.Error("namespace must be <db>.<rp>")
        }
    }

INSERT_STATEMENT:
    INSERT INTO NAMESPACE LINE_PROTOCOL
    {
        stmt := &InsertStatement{}
        stmt.LineProtocol = $4

        if len($3) != 2 {
            QLlex.Error("namespace must be <db>.<rp>")
        } else {
            stmt.DB = $3[0]
            stmt.RP = $3[1]
            $$ = stmt
        }
    }
    |INSERT INTO LINE_PROTOCOL
    {
        stmt := &InsertStatement{}
        stmt.LineProtocol = $3
        $$ = stmt
    }
    |INSERT LINE_PROTOCOL
    {
        stmt := &InsertStatement{}
        stmt.LineProtocol = $2
        $$ = stmt
    }

NAMESPACE:
    IDENT
    {
        $$ = []string{$1}
    }
    |IDENT DOT NAMESPACE
    {
        ns := []string{$1}
        $$ = append(ns, $3...)
    }

LINE_PROTOCOL:
    TIME_SERIE
    {
        $$ = $1
    }
    |TIME_SERIE TIME
    {
        $$ = $1 + " " + $2
    }

TIME_SERIE:
    MEASUREMENT COMMA KV_RAWS KV_RAWS
    {
        $$ = $1 + $2 + $3 + " " + $4
    }

KEY_VALUE:
    IDENT EQ IDENT
    {
        p := NewPair($1, $3)
        $$ = *p
    }
    |IDENT EQ STRING
    {
        p := NewPair($1, $3)
        $$ = *p
    }
    |IDENT EQ INTEGER
    {
        p := NewPair($1, $3)
        $$ = *p
    }
    |IDENT EQ DECIMAL
    {
        p := NewPair($1, $3)
        $$ = *p
    }

KEY_VALUES:
    KEY_VALUE
    {
        $$ = Pairs{$1}
    }
    |KEY_VALUE COMMA KEY_VALUES
    {
        $$ = append($3, $1)
    }

KV_RAWS:
    KV_RAW
    {
        $$ = $1
    }
    |KV_RAW COMMA KV_RAWS
    {
        $$ = $1 + $2 + $3
    }

KV_RAW:
    IDENT EQ RAW
    {
        $$ = $1 + $2 + $3
    }

MEASUREMENT:
    IDENT
    {
        $$ = $1
    }

TIME:
    INTEGER
    {
        $$ = strconv.FormatInt($1, 10)
    }
%%