# Raycast登録・検証記録

## 登録

1. GitHub CLIで認証し、`gh mutual-follow --help` が動く状態にする。
2. Raycastの設定から **Extensions → Script Commands → Add Script Directory** を開く。バージョンによって設定画面の配置は異なる。
3. このリポジトリをチェックアウトした場所の `scripts/raycast` を選ぶ。
4. Raycastで **GitHub Mutual Follow** を検索する。実行すると実アカウントのフォロー関係が変更される。

登録方法とメタデータは[Raycast公式リポジトリ](https://github.com/raycast/script-commands)に準拠する。

## 実装と制限

- `unfollow --all --yes` → `follow-back --all --yes` の順に実行する。
- 解除の失敗時は後続を実行しない。フォローバックの失敗も終了コードを伝播する。
- 結果はRaycast内に表示する。認証情報をスクリプトに記載しない。
- シェルの対話設定を読み込まず、既存のPATHに `/opt/homebrew/bin` と `/usr/local/bin` を追加する。独自の場所にある `gh` はPATHの調整が必要。
- 除外設定・自動再試行・巻き戻し・複数起動間の排他制御はない。
- 2026-11-11に、利用者からRaycastへの登録と実行が成功したとの報告を受けた。自動検証ではモックのみを使い、実アカウントを変更しない。

## TDDと検証

- `scripts/test-raycast.sh` を先に追加し、スクリプト未実装による失敗（期待終了コード0、実際127）を確認した。
- テストは一時ディレクトリの `gh` モックのみを使用し、実アカウント・認証情報にアクセスしない。
- 成功時の引数と順序、解除失敗時の停止、フォローバック失敗時の終了コード、結果出力を検証する。
- `make check` に組み込み、既存のCIとpre-commitでも実行する。
- `make check` 成功（整形、vet、race付きGoテスト、Raycastモック3ケース、ビルド）。
- `make security` 成功（Gitleaks: 秘密情報なし、govulncheck: 脆弱性なし）。最初はサンドボックスのキャッシュ・ネットワーク制限で失敗したため、権限付きで両方を再実行して成功を確認した。
- `bash -n` と `git diff --check` 成功。インストール済み拡張の `gh mutual-follow --help` でコマンドとフラグを確認した。

### 2026-11-11 PR作成時の再検証

- 最新の `origin/main`（`c3f5aa3`）を取り込み、既存のリリース処理のモックテストとRaycastのモックテストが両方 `make check` で実行されるよう統合した。
- ローカルのGo 1.27.3では `make check` が成功したが、`make security` は標準ライブラリの `GO-2026-6859` を検出して失敗した。
- 検査が示した修正版Go 1.27.4を指定し、`GOTOOLCHAIN=go1.27.4 make check security` が成功した。検査の無効化や検出結果の抑制は行っていない。Go 1.27.3を使う環境ではツールチェーンの更新が必要。
- `bash -n scripts/raycast/github-mutual-follow.sh scripts/test-raycast.sh` と `git diff --check` が成功した。
- Raycastでの登録・実行成功は利用者による確認。PR準備中の検証では実アカウントの変更を行っていない。
