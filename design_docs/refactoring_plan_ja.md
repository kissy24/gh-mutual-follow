# gh-mutual-followリファクタリング計画書

## 1. はじめに

本文書は、CLIツール `gh-mutual-follow` のリファクタリング計画について概説するものです。このツールは、GoとBubble Teaフレームワークで構築されたTUIアプリケーションであり、ユーザーがGitHub上の相互フォロー関係を管理するのを支援します。

このリファクタリングの主な目的は、現在のコードベースに存在するアーキテクチャ上の問題を解決し、アプリケーションの**保守性**、**テスト容易性**、そして全体的な**コード品質**を向上させることです。

## 2. 現状のアーキテクチャと課題

現在の実装は機能しているものの、いくつかのアーキテクチャ上の欠点があります。

-   **モノリシックな `main.go`**: TUIアプリケーションの全ロジック（モデル、Updateループ、ビューのレンダリング、スタイル定義、非同期コマンドの起動など）が、300行を超える単一の `main.go` ファイルに詰め込まれています。これはコードの可読性、保守性を著しく低下させています。
-   **責務の密結合**: ビューのロジック（UIコンポーネント、スタイル）とビジネスロジック（GitHubアクションの呼び出し）が強く結合しています。特に `Update` 関数がすべてを処理しており、単一責任の原則に違反しています。
-   **TUIのテストが不在**: `internal/cli` パッケージにはテストが存在しますが、TUIロジック自体は全くテストされていません。モノリシックな構造が、UIの振る舞いに対する単体テストの作成を非常に困難にしています。
-   **`cli` パッケージの不完全な抽象化**: `internal/cli` パッケージは、テストのためにグローバルな `runCommand` 変数に依存しています。より堅牢なアプローチは、インターフェースによる依存性注入を利用することです。これにより、アプリケーションロジックとシェルコマンドの具象実装を分離できます。

## 3. リファクタリング提案

これらの課題に対処するため、以下の変更を提案します。

### 提案1: TUIロジックを専用パッケージに分離する

`main.go` の内容を分解し、責務ごとに専用の `internal/tui` パッケージへ整理します。

-   **`internal/tui/model.go`**: コアとなるBubble Teaアプリケーション（`Model` 構造体とその `Init`, `Update`, `View` メソッド）を配置します。
-   **`internal/tui/styles.go`**: すべての `lipgloss` スタイル定義（`TUIStyles`, `defaultStyles`）をこのファイルに移動し、関心事を分離します。
-   **`internal/tui/components.go`**: `itemDelegate` のようなUIコンポーネント固有のロジックをここに配置します。
-   **`internal/tui/commands.go`**: `loadDataCmd` のような非同期処理を開始する `tea.Cmd` 関数をこのファイルにまとめます。
-   **`main.go`**: ルートの `main.go` は、`tui` パッケージで定義されたTUIプログラムを初期化して実行するだけのシンプルなエントリーポイントになります。

### 提案2: GitHubクライアントを抽象化する

GitHub CLIとの対話を担う責務をインターフェースに抽象化します。

-   **`internal/cli` を `internal/github` にリネーム**: パッケージの責務をより正確に反映する名前に変更します。
-   **`Client` インターフェースの定義**: GitHubとのやり取りの契約を定義するために `github.Client` インターフェースを作成します。

    ```go
    package github

    type Client interface {
        GetUser() (string, error)
        GetFollowing(user string) ([]string, error)
        GetFollowers(user string) ([]string, error)
        Unfollow(user string) error
        Follow(user string) error
    }
    ```

-   **`ghClient` の実装**: 既存のコマンド実行ロジックを、`Client` インターフェースを実装する `ghClient` 構造体にカプセル化します。
-   **依存性の注入 (Dependency Injection)**: TUIの `Model` は、具象実装ではなく `github.Client` インターフェースに依存するようにします。これにより、TUIと `gh` コマンドラインツールが疎結合になり、実装の差し替えやテストのためのモック化が容易になります。

### 提案3: TUIのテストを追加する

改善されたアーキテクチャを活かし、TUIの単体テストを導入します。

-   **`internal/tui/tui_test.go`**: 新しいテストファイルを作成します。
-   **`Update` 関数のテスト**: `Update` 関数が、様々な `tea.Msg` 入力に対してモデルの状態を正しく変更することを検証するテストを記述します。
-   **GitHubクライアントのモック化**: `github.Client` インターフェースを利用して `mockClient` を作成し、様々なAPIレスポンス（成功、エラー、特定のデータ）をシミュレートします。これにより、実際のAPIコールなしにTUIの反応をテストできます。

## 4. リファクタリング後のディレクトリ構造

リファクタリング後、プロジェクトの構造は以下のようになります。

```
.
├── main.go
├── go.mod
├── go.sum
└── internal/
    ├── github/      <-- (cliからリネーム)
    │   ├── client.go    (ClientインターフェースとghClient実装)
    │   └── client_test.go
    └── tui/
        ├── model.go     (コアとなるBubble Teaモデル、Init, Update, View)
        ├── components.go(itemDelegateなど)
        ├── styles.go    (lipglossスタイル定義)
        ├── commands.go  (tea.Cmdを生成するファクトリ関数)
        └── tui_test.go  (Update関数の単体テスト)
```

## 5. 実施計画

リファクタリングは以下のフェーズで実施します。

1.  **準備**: `design_docs` ディレクトリを作成し、本計画書を保存します。
2.  **クライアントの抽象化**: `internal/cli` を `internal/github` にリネームし、`Client` インターフェースを定義して既存のコードをリファクタリングします。
3.  **TUIの疎結合化**: TUIの `Model` が `github.Client` インターフェースに依存するように修正します。
4.  **TUIの関心事の分離**: `main.go` からTUIロジックを提案1で述べたように `internal/tui` パッケージに移動します。
5.  **`main.go`の簡素化**: `main.go` をシンプルなアプリケーションエントリーポイントに整理します。
6.  **テストの追加**: モックのGitHubクライアントを使用して、TUIの `Update` 関数の単体テストを実装します。
