# gh-mutual-follow

GitHubの片方向のフォロー関係を確認し、フォローバック・フォロー解除を行うGitHub CLI拡張です。TUIの操作はなく、コマンドで目的を直接指定します。

```sh
gh mutual-follow list
gh mutual-follow follow-back --all
gh mutual-follow unfollow --all --exclude alice --dry-run
```

## 導入

対応環境はmacOS / Linux（amd64・arm64）、GitHub.comです。GitHub CLIと、その認証情報を利用します。GitHub CLI 2.101.0でローカル拡張の動作を確認しています。それ以前のバージョンは初版の検証対象外です。

```sh
gh auth login --hostname github.com
```

バイナリを添付した最初のリリース公開後は、以下でインストールできます。

```sh
gh extension install kissy24/gh-mutual-follow
gh extension upgrade mutual-follow
```

リリース前、またはソースから利用する場合は、Go 1.26以上でこのリポジトリのルートからビルドします。

```sh
go build -o gh-mutual-follow .
gh extension install .
gh mutual-follow --help
```

ローカル登録は実行ファイルを参照するため、コードを更新したら再ビルドしてください。削除は `gh extension remove mutual-follow` です。

## 使い方

### 関係を確認する

```sh
# 片方向の関係を両方表示
gh mutual-follow list

# 自分だけがフォロー中
gh mutual-follow list --type following-only

# 相手だけがフォロー中
gh mutual-follow list --type followers-only
```

アカウント、件数、ログイン名、プロフィールURLを表示します。全ページを取得し、全件をログイン名順に出力します。相互フォロー中の相手は対象外です。引数なしの `gh mutual-follow` はヘルプを表示し、通信しません。

### フォローバック・解除する

```sh
# 個別にフォローバック
gh mutual-follow follow-back bob

# 未フォローバックの全員に実行
gh mutual-follow follow-back --all

# 指定した相手を解除
gh mutual-follow unfollow alice charlie

# フォローを続けたい相手を除いて、まず予定を確認
gh mutual-follow unfollow --all --exclude alice --exclude charlie --dry-run

# 確認入力を省略して実行
gh mutual-follow unfollow --all --exclude alice --yes
```

通常は対象者全員と件数を表示し、1回だけ `[y/N]` で確認します。`y` または `yes` で実行し、空入力・否認・EOFで中止します。

| 指定 | 動作 |
| --- | --- |
| `USER...` | 指定した相手だけに実行。名前の大文字小文字は区別しない |
| `--all` | 該当区分の全員に実行。`USER...` との併用不可 |
| `--exclude USER` | 対象から除外。複数回指定可能 |
| `--dry-run` | 予定の表示のみ。`--yes` を併用しても変更しない |
| `--yes`, `-y` | 確認を省略。非端末入力での変更には必須 |
| `--json` | JSON出力。変更時は `--yes` または `--dry-run` が必要 |

明示した相手が該当区分にいない場合、1件も変更せずエラーにします。相互フォロー中の相手は解除できません。全員が除外された場合や候補0人の場合は、確認せず正常終了します。

除外はそのコマンドだけに適用されます。繰り返し使う場合は、シェルの関数やスクリプトにコマンドを保存できます。

### JSON・終了コード

```sh
gh mutual-follow list --json
gh mutual-follow unfollow --all --dry-run --json
gh mutual-follow follow-back --all --yes --json
```

JSONは標準出力に1つのオブジェクトを返します。`schema_version` は `1`。`account`、`command`、`status`、`error`、`users`、`results`、`summary` を含み、使わない配列も `[]` で返します。予定・成功・失敗・結果不明・未着手・除外を区別します。詳細は[仕様書](docs/cli-spec.md)を参照してください。

| 終了コード | 意味 |
| --- | --- |
| `0` | 成功、ドライラン、対象0人、ヘルプ |
| `1` | 認証・通信・変更・出力などの実行失敗。部分成功・結果不明を含む |
| `2` | 引数・対象エラー、必要な `--yes` の不足 |
| `3` | 確認で中止 |
| `130` | Ctrl+Cで中断 |

## 認証と実行結果

認証情報の優先順位は `GH_TOKEN` → `GITHUB_TOKEN` → `gh auth token --hostname github.com` です。トークンはメモリ上でのみ扱い、画面やファイルに出力しません。接続先はGitHub.comに固定され、カレントディレクトリのGit設定に依存しません。

変更には、OAuth / classic PATなら `user:follow` が必要です。GitHub CLIのOAuth認証の場合は、必要に応じて以下を実行します。

```sh
gh auth refresh --hostname github.com --scopes user:follow
```

fine-grained PATはFollowersのread権限で一覧、write権限で変更を行います。環境変数のトークンを使っている場合は、そのトークンの権限を更新してください。[GitHub APIの権限仕様](https://docs.github.com/en/rest/users/followers)

全件の取得後に対象を確定し、直列で処理します。最初の失敗・レート制限で停止し、成功した変更を巻き戻しません。各リクエストのタイムアウトは30秒で、自動再試行はしません。通信切断や中断で変更の反映を確認できなければ「結果不明」と表示します。再実行の前に `list` で現在の関係を確認できます。

GitHubの関係データは取得中や確認待ちにも変化し得ます。ドライランは実行時点の予定であり、次回と同じ対象になることを保証するものではありません。

## 開発・検証

```sh
make check             # gofmt、go vet、全テスト（race付き）、ビルド
make security          # Gitleaksによる秘密情報検査、govulncheck
make workflow-check    # GitHub Actionsの検証
make release-check     # 4種類の配布バイナリを生成
bash scripts/smoke-extension.sh # 隔離設定でgh拡張の起動確認
```

検査ツールはMakefile内の固定バージョンを `go run` で取得します。初回取得と脆弱性データ更新にはネットワークが必要です。

`pre-commit` をインストールした環境では、以下でコミット前検査を有効にします。

```sh
make hooks
pre-commit run --all-files
```

GitHub ActionsはPR・mainへのpush・週次でテストとセキュリティ検査を実行します。テストはmacOS / Linux、Go 1.26系 / stableの組み合わせです。バージョンタグ `v*` のpush時は、同じ検査の成功後にmacOS / Linuxのamd64・arm64バイナリとSHA-256チェックサムを公開します。`-rc.1` 等を含むタグはプレリリースになります。

必須チェックの設定は[セキュリティ・CI運用](docs/security-and-ci.md)を参照してください。

- [開発ルール](AGENTS.md)
- [現行CLI仕様](docs/cli-spec.md)
- [実装・TDD・検証記録](docs/implementation.md)
- [セキュリティ・CI運用](docs/security-and-ci.md)

`design_docs/` は旧TUIの設計履歴、`downloads/` はCLI移行前の仕様案です。現行仕様は `docs/cli-spec.md` を参照してください。
