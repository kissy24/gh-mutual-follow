# リリースCIの再実行

2026-11-03: Release run 36432494244はv1.0.0のリリースが既に存在するため失敗していた。

公開処理をscripts/publish-release.shへ切り出し、同じタグのリリースが存在するときは既存の公開物を保持して成功する。未公開タグは従来どおりビルド済みassetを公開する。事前確認や公開時のエラーは、作成処理の失敗として伝播する。

scripts/test-publish-release.shはghをモックし、既存リリースの保持、新規公開、prerelease、公開失敗の伝播を検証する。make checkにも組み込み、実アカウントに公開せずCIで確認する。

ローカル検証: make check (race検出付き全テスト・ビルド・公開モックテスト)、make security (Gitleaks / govulncheck)成功。
