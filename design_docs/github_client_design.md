# GitHub CLI 連携機能

本ドキュメントは `gh-mutual-follow` アプリケーションにおける GitHub CLI との連携機能に関する設計の経緯を記述するものです。

## 機能概要

`internal/cli` パッケージ（リファクタリング後は `internal/github`）は、`gh` コマンドラインツールをラップし、GitHub APIとの通信を担います。これにより、アプリケーションはユーザーのフォロー/フォロワー情報を取得し、フォロー/アンフォロー操作を実行します。

## 実装内容

`gh` CLI ツールとの連携のため、`internal/cli/cli.go` に以下の関数が実装されました。

-   `GetUser()`: 認証済み GitHub ユーザー名を取得します。
-   `GetFollowing(user string)`: 指定ユーザーがフォローしているユーザーリストを取得します。
-   `GetFollowers(user string)`: 指定ユーザーをフォローしているユーザーリストを取得します。
-   `Unfollow(user string)`: 指定ユーザーをアンフォローします。
-   `Follow(user string)`: 指定ユーザーをフォローします。
-   `GetMutualFollowsData(authenticatedUser string, following, followers []string)`: 相互フォローでないユーザーのリスト（「Following」と「Followers」）を生成します。

### 機能改善

-   **ページネーション対応**: 当初、APIから取得できるユーザー数に限りがありましたが、`gh api` コマンドに `--paginate` フラグを追加することで、全フォロワー/フォロー中ユーザーを取得できるように修正されました。
-   **エラーメッセージの詳細化**: `runCommand` ヘルパー関数を修正し、`gh` CLI コマンドの実行が失敗した際に、`stderr` の内容をエラーメッセージに含めるように変更しました。これにより、特に認証トークンの権限不足（スコープの問題）などのデバッグが容易になりました。

## テスト

`internal/cli/cli_test.go` に、`runCommand` 関数をモック化した単体テストが実装され、各関数の動作が検証されています。
