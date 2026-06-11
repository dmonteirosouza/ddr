#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

version="${1:-}"
if [[ -z "$version" ]]; then
  version="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go nao encontrado. Instale em https://go.dev/dl/ e tente de novo."
  exit 1
fi

dist_dir="$repo_root/dist"
work_dir="$repo_root/.release"
rm -rf "$dist_dir" "$work_dir"
mkdir -p "$dist_dir" "$work_dir"

export GOCACHE="${GOCACHE:-$work_dir/go-cache}"
mkdir -p "$GOCACHE"

build_package() {
  local arch="$1"
  local package="ddr_${version}_darwin_${arch}"
  local package_dir="$work_dir/$package"

  mkdir -p "$package_dir"

  CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" go build \
    -ldflags "-s -w -X ddr/internal/app.Version=$version" \
    -o "$package_dir/ddr" ./cmd/ddr

  cat >"$package_dir/install.command" <<'INSTALLER'
#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

install_dir="$HOME/.local/bin"
mkdir -p "$install_dir"

cp ./ddr "$install_dir/ddr"
chmod +x "$install_dir/ddr"

path_line='export PATH="$HOME/.local/bin:$PATH"'
shell_rc="$HOME/.zshrc"
if ! grep -qs 'HOME/.local/bin' "$shell_rc" 2>/dev/null; then
  {
    echo ""
    echo "# ddr"
    echo "$path_line"
  } >>"$shell_rc"
fi

export PATH="$install_dir:$PATH"

echo ""
echo "ddr instalado com sucesso."
echo "Local: $install_dir/ddr"
echo ""
"$install_dir/ddr" version || true
echo ""
echo "Para testar depois, abra um novo Terminal e rode:"
echo "  ddr scan"
echo ""
read -r -p "Pressione Enter para fechar..."
INSTALLER

  chmod +x "$package_dir/install.command"

  cat >"$package_dir/LEIA-ME.txt" <<README
ddr $version para macOS ($arch)

Instalacao facil:
1. De duplo clique em install.command.
2. Se o macOS pedir confirmacao, escolha Abrir.
3. Abra um novo Terminal e rode:
   ddr scan

O instalador copia o binario para:
  ~/.local/bin/ddr

Ele tambem garante que ~/.local/bin esteja no PATH do zsh.

Para remover:
  rm ~/.local/bin/ddr

Observacao:
Volumes do Docker nunca sao apagados automaticamente pelo ddr.
README

  if command -v zip >/dev/null 2>&1; then
    (cd "$work_dir" && COPYFILE_DISABLE=1 zip -qr "$dist_dir/$package.zip" "$package")
  elif command -v ditto >/dev/null 2>&1; then
    (cd "$work_dir" && ditto -c -k --norsrc --keepParent "$package" "$dist_dir/$package.zip")
  else
    echo "Nem ditto nem zip foram encontrados para criar o pacote."
    exit 1
  fi
}

build_package arm64
build_package amd64

echo "Pacotes gerados em:"
ls -1 "$dist_dir"
