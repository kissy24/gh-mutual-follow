# CLI移行のPR

更新日: 2026-09-28

## ブランチ運用

GitHub Flowに沿って、`main` から `feat/cli-extension` を作成し、`main` をベースにPRを提出する。変更内容はPRでレビューし、CIの成功を確認してからマージする。今回の依頼はPR作成までであり、マージ・タグ作成・リリース公開は行わない。

開始時のローカル・リモート `main` はともに `ea378a5`。既存のオープンPRはなく、今回のCLI実装・テスト・CI・ドキュメントを1つの変更として提出する。

## 提出内容

- TUIからGitHub CLI拡張への置き換え。
- 対象検証、確認、除外、ドライラン、JSON、途中失敗・中断の処理。
- TDDで追加したテスト、GitHub Actions、pre-commit、セキュリティ検査、リリース処理。
- README、開発ルール、現行仕様、対応記録、副作用のないデモ。

検証は `make check`、`make security`、ワークフロー検査をコミット時のpre-commitで実行する。GitHub Actionsの結果はPRのChecks欄で確認する。実アカウントのフォロー関係は変更しない。
