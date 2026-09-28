# セキュリティとCI運用

更新日: 2026-09-27

## 実装で守ること

- 対象の全件取得・検証に成功するまで変更を開始しない。
- 認証はGitHub CLIと同じ優先順位で解決し、実行中は同じトークンを使う。トークンはファイル・ログ・引数へ出力しない。
- API通信先は `https://api.github.com` に固定する。HTTPリダイレクトを追従せず、ページネーションの外部URLや別エンドポイントを拒否する。
- ログイン名を検証し、制御文字・パス・オプションの注入を拒否する。認証コマンドはシェルを介さない引数配列で実行する。
- 応答本文と外部コマンドのエラーをそのまま表示しない。HTTPステータスから安全なメッセージを構築する。
- 1リクエスト30秒、応答は最大8 MiB。変更の自動再試行をせず、失敗時に後続を停止する。
- 確認の表示に失敗した場合や、非端末で `--yes` がない場合は変更を開始しない。
- 認証情報を必要とする実アカウントの変更は、通常のテストやCIに含めない。

## 自動検査

| 検査 | ローカル | GitHub Actions |
| --- | --- | --- |
| 整形・静的解析 | `make check` | 全テストジョブ |
| テスト・race検出 | `make check` | macOS/Linux × Go 1.26系/stable |
| 作業ツリーの秘密情報 | `make security` | Securityジョブ |
| コミット対象の秘密情報 | `make staged-secrets` | pre-commit。CIではGit履歴全体も検査 |
| 依存・標準ライブラリの既知脆弱性 | `make security` | Securityジョブ、週次実行 |
| ワークフロー構文 | `make workflow-check` | Securityジョブ |
| 4種類の配布バイナリ | `make release-check` | Securityジョブ |
| ローカル拡張の起動 | `bash scripts/smoke-extension.sh` | 全テストジョブ |

Goの脆弱性検査には公式の[govulncheck](https://go.dev/doc/security/vuln/)を使用する。検査ツールはMakefileのバージョンに固定し、Gitleaksは既定ルールを有効にする。除外は生成バイナリ・カバレッジ等に限定する。

CIは `pull_request` を使用し、外部PRコードに書き込みトークンやリポジトリの秘密情報を渡さない。通常の権限は `contents: read`。リリースジョブだけ `contents: write` とし、テスト・セキュリティ検査の成功を依存条件にする。

外部ActionはコミットSHA固定、checkoutの認証情報永続化は無効。DependabotでGoモジュールとGitHub Actionsを週次確認する。Makefile内の検査ツールの固定バージョンは更新PRで手動確認し、定期的に更新する。

## pre-commitの有効化

```sh
make hooks
pre-commit run --all-files
```

`pre-commit` 本体とGoが必要。フックは整形を勝手に書き換えず、問題があれば失敗する。Goファイルは `gofmt -w` で修正し、検査をやり直す。検査失敗を `--no-verify` 等で恒常的に回避しない。

## GitHub側の必須チェック

ワークフローファイルだけではmainへの未検証マージを禁止できない。リポジトリ管理者はRulesetsまたはBranch protectionで、PR必須と以下のステータスチェック必須を設定する。

- `Test (ubuntu-latest, Go 1.26.x)`
- `Test (ubuntu-latest, Go stable)`
- `Test (macos-latest, Go 1.26.x)`
- `Test (macos-latest, Go stable)`
- `Security`

チェック名はGitHub上の初回実行結果で確認して選択する。これらのリモート設定はコードとして追加しただけでは有効にならず、今回のローカル実装では変更していない。

## 配布と限界

タグ `vMAJOR.MINOR.PATCH` のpushで、CI成功後にバイナリとSHA-256チェックサムを公開する。配布アセットは[公式拡張の命名規則](https://github.com/cli/gh-extension-precompile#extensions-written-in-other-compiled-languages)に従う。

GitHub側の関係は取得後にも変化する。取得・確認・更新を1つのトランザクションにはできないため、最新状態の確認には `list` を再実行する。タイムアウトやキャンセルはサーバー側の変更取り消しを保証しない。結果不明の対象を自動で再実行しない。
