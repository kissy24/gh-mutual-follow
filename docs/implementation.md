# CLI移行の実装・検証記録

更新日: 2026-09-27

## 変更の目的

TUIの起動・ペイン切り替え・カーソル操作をなくし、GitHub CLI拡張のコマンドからフォロー関係を整理できるようにした。別リポジトリの新規開発案から、既存リポジトリを置き換える方針に更新した。

## 実装内容

- `internal/tui` とBubble Tea / Lipgloss等の実行時依存を削除した。
- `internal/domain` にIDによる差分、本人除外、重複排除、名前の検証、対象選択・除外を分離した。
- `internal/cli` に `list`、`follow-back`、`unfollow`、確認、ドライラン、JSON、終了コード、中断・部分失敗の集計を実装した。
- `internal/github` を認証ユーザーJSONとHTTP APIを扱うクライアントに置き換えた。100件ずつ全ページを取得し、読み取りが完了するまで変更を始めない。
- 通信はGo標準HTTPクライアント、認証はGitHub CLIの環境変数または `gh auth token` で解決する。トークンを実行中に固定するため、確認したアカウントと変更の認証を一致させる。
- HTTP状態に基づくエラー分類、30秒タイムアウト、Ctrl+C、結果不明、最初の失敗での停止を実装した。
- APIのリダイレクト拒否、ページリンク検証、制御文字やパスの拒否、秘密情報を含み得る生エラーの非表示を実装した。
- `AGENTS.md` にTDD・Green・セキュリティ自動化・docs記録・README同期を継続ルールとして追加した。
- `Makefile`、pre-commit、GitHub Actions、Dependabot、リリースビルドを追加した。

## TDDの記録

1. 対象判定のテストを先に追加し、`User` / `Difference` / `Select` 未定義でRedを確認した。実装後にGreenになった。
2. CLIのコマンド契約テストを先に追加し、`Run` / `Report` 未定義でRedを確認した。引数検証、ドライラン、確認、部分失敗、中断を実装してGreenになった。
3. HTTPクライアントのテストを先に置き換え、旧インターフェースでは新しい認証・ページ取得・変更契約を満たせずRedになることを確認した。クライアントの置き換え後にGreenになった。
4. フィルター指定時に対象外区分を0件と表示してしまう問題をテストで再現した。通常出力を修正し、取得失敗を「空の一覧」と見せないテストもGreenにした。
5. race検査でテスト用HTTPサーバーのカウンター競合を検出した。atomicカウンターに変更し、race検査をGreenにした。
6. CLIからHTTPまで通した回帰テストを追加し、後続ページの取得失敗で変更が始まらないこと、相互フォローを除いた相手だけを解除することを検証した。
7. 実際のGitHub CLIでローカル拡張の登録を検証し、絶対パス指定を `gh extension install .` に修正した。隔離設定でインストール・起動・削除まで成功した。

## 検証結果

ローカル環境: macOS arm64、Go 1.27.1、GitHub CLI 2.101.0。

| 検査 | 結果 |
| --- | --- |
| `make check` | Green。整形、vet、全テスト、race検査、ビルド成功 |
| CLIテストのステートメントカバレッジ | 91.6% |
| 対象判定のステートメントカバレッジ | 97.8% |
| GitHub通信のステートメントカバレッジ | 84.4% |
| Gitleaks作業ツリー検査 | 検出なし |
| govulncheck | 既知脆弱性の検出なし |
| actionlint | エラーなし |
| `make release-check` | darwin/linux × amd64/arm64の4バイナリを生成 |
| 隔離した `gh` の拡張起動 | 登録、ヘルプ、バージョン、削除が成功 |
| pre-commit導入 | この作業コピーの `.git/hooks/pre-commit` に導入済み |
| `pre-commit run --all-files` | 全4フックがPassed |
| `GOTOOLCHAIN=go1.26.0 go test ./...` | 最低Goバージョンでも全テストGreen |
| Gitleaks Git履歴検査 | 8コミットを検査し、検出なし |

テストではモックまたはローカルHTTPサーバーを使用した。実アカウントのフォロー関係は変更していない。

## 自動化と公開前の確認

- PR・mainへのpush・週次にGitHub Actionsで品質・セキュリティ検査を実行する。
- バージョンタグによるリリースは、同じCIワークフローが成功した後だけ実行する。
- 公開アセットはGitHub CLIが認識するOS・CPUサフィックスとSHA-256チェックサムを付ける。
- 初回実装時点ではpush・タグ作成・GitHub Releasesへの公開は行っていない。後続のPR作成・CI確認は `pull-request.md` に記録する。実リリースからのインストール・アップグレードは公開時に確認する。
- 他OS/CPUはクロスビルドを検証した。LinuxやmacOS Intelでの実機実行はこのローカル環境では行っていない。
- リポジトリのRulesets / Branch protectionは変更していない。必須チェックの設定方法は `security-and-ci.md` に記載した。
- `downloads/` の原案は履歴として保持し、現行仕様を `docs/cli-spec.md`、利用手順をREADMEにまとめた。
