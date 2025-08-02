---
title: インストール
description: システムにProbeをインストールする方法を学ぶ
weight: 10
---

# インストール

Probeは軽量でシングルバイナリのツールで、複数の方法でインストールできます。お使いの環境に最適な方法を選択してください。

## システム要件

- **オペレーティングシステム**: Linux、macOS、Windows
- **アーキテクチャ**: amd64、arm64
- **依存関係**: なし（静的リンクされたバイナリ）

## インストール方法

### 1. プリビルドバイナリのダウンロード

Probeをインストールする最も簡単な方法は、GitHubのリリースページからプリビルドバイナリをダウンロードすることです。

1. [Probeリリースページ](https://github.com/linyows/probe/releases)にアクセス
2. お使いのシステムに適したバイナリをダウンロード：
   - **Linux amd64**: `probe-linux-amd64`
   - **Linux arm64**: `probe-linux-arm64`
   - **macOS amd64**: `probe-darwin-amd64`
   - **macOS arm64**: `probe-darwin-arm64`
   - **Windows amd64**: `probe-windows-amd64.exe`

3. バイナリを実行可能にする（Linux/macOS）：
   ```bash
   chmod +x probe-linux-amd64
   ```

4. PATHが通ったディレクトリに移動：
   ```bash
   sudo mv probe-linux-amd64 /usr/local/bin/probe
   ```

### 2. Goでインストール

Go 1.19以降がインストールされている場合、Probeを直接インストールできます：

```bash
go install github.com/linyows/probe/cmd/probe@latest
```

これにより、`probe`バイナリが`$GOPATH/bin`ディレクトリにインストールされます。

### 3. ソースからビルド

Probeをソースからビルドするには：

```bash
git clone https://github.com/linyows/probe.git
cd probe
go build -o probe ./cmd/probe
sudo mv probe /usr/local/bin/
```

### 4. Docker

DockerコンテナでProbeを実行：

```bash
docker run --rm -v $(pwd):/workspace linyows/probe:latest /workspace/workflow.yml
```

## インストールの確認

インストール後、Probeが正しく動作していることを確認します：

```bash
probe --version
```

以下のような出力が表示されるはずです：
```
Probe Version v1.0.0 (commit: abc123)
```

## 次のステップ

Probeのインストールが完了したら、次の内容に進みましょう：

1. **[最初のワークフローを作成](../quickstart/)** - 簡単な例から始める
2. **[基本を学ぶ](../understanding-probe/)** - 核となる概念を理解する
3. **[例を探る](../../tutorials/)** - 実践的な使用例を見る

## トラブルシューティング

### Permission Denied

Linux/macOSで「permission denied」エラーが出る場合：

```bash
chmod +x probe
```

### Command Not Found

`probe`コマンドが見つからない場合、バイナリがPATHに含まれていることを確認します：

```bash
echo $PATH
which probe
```

### Apple Silicon上のARM64

Apple Silicon Mac（M1/M2）では、パフォーマンス向上のため`darwin-arm64`バイナリを使用してください。