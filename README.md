# MeC Livros Ebook Downloader

Aplicativo em Go 1.26 para download de ebooks da plataforma [MeC Livros](https://meclivros.mec.gov.br).

---

## 🚨 ⚠️ AVISO DE SEGURANÇA - LEIA ANTES DE USAR

**O token JWT é uma credencial sensível que concede acesso total à sua conta!**

### ⛔ NÃO FAÇA:

- ❌ Nunca compartilhe seu token com ninguém
- ❌ Não publique em repositórios públicos, issues, pull requests ou discussões
- ❌ Não envie por email, chat, SMS ou qualquer outro meio
- ❌ Não reutilize em outros aplicativos ou plataformas
- ❌ Não faça commit do arquivo `~/.mec_livros_token` em repositórios

### 🔓 RISCOS DE UM TOKEN COMPROMETIDO:

- Qualquer pessoa com seu token pode acessar **todos os seus ebooks alugados**
- Fazer download ilimitado durante o período de aluguel
- Acessar seu histórico de leitura e preferências
- Potencialmente usar dados pessoais vinculados à sua conta gov.br

---

## Estrutura

```
cmd/mectlivros/              # Ponto de entrada
internal/
├── cache/                   # Cache JWT
├── downloader/              # Download paralelo
└── epub/                    # Geração de EPUB
pkg/models/                  # Modelos de dados
```

## Instalação

Requisitos: Go 1.26+ e token JWT válido.

### Via `go install` (Recomendado)

```bash
go install github.com/publi0/mec-livros-epub/cmd/mectlivros@latest
mectlivros
```

O binário será instalado em `$GOPATH/bin/` (geralmente `~/go/bin/`).

### Via Build Local

```bash
git clone https://github.com/publi0/mec-livros-epub.git
cd mec-livros-epub
go build -o mectlivros ./cmd/mectlivros
./mectlivros
```

### Via `go run`

```bash
go run ./cmd/mectlivros
```

## Uso

1. Informa ou usa token do cache (`~/.mec_livros_token`)
2. Lista ebooks alugados
3. Seleciona qual baixar
4. Download paralelo com 8 workers
5. Gera EPUB pronto para leitura

## Dependências

O projeto utiliza apenas **dependências externas mínimas**:

```bash
go mod download
```

Verifique as dependências:

```bash
go list -m all
```

Para atualizar dependências:

```bash
go get -u ./...
go mod tidy
```

## Configuração

- Workers: 8
- Timeout: 30s por requisição, 10min total
- Cache: `~/.mec_livros_token` (permissão 600 - apenas proprietário)

## Recursos Modernos (Go 1.26)

- **slog**: Structured logging nativo
- **context**: Cancellation e timeouts
- **cmp.Or**: Generic comparisons
- **atomic**: Thread-safe counters
- **goroutines**: Download paralelo

## Obter Token JWT

1. Acesse [meclivros.mec.gov.br](https://meclivros.mec.gov.br)
2. Faça login com gov.br
3. DevTools (F12) > Application > Local Storage
4. Copie o valor do campo `token` ou `jwt`
5. Cole no aplicativo quando solicitado

⚠️ **Lembre-se**: Use o token apenas nesta máquina local. Não o compartilhe.

## Cache

Token é salvo automaticamente em `~/.mec_livros_token` para reutilização futura.

**Proteção de arquivo**: O cache é criado com permissão 600 (apenas leitura/escrita do proprietário). Não altere essas permissões.

Para limpar:

```bash
rm ~/.mec_livros_token
```

Ou use o aplicativo:

```
Usar token do cache? [Y/n/limpar]: limpar
```

## Saída Exemplo

```
============================================
📚 MEC LIVROS - EBOOK DOWNLOADER (Go 1.26)
============================================

🔐 Token em cache encontrado
   Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

📚 Encontrados 1 ebook(s):

   [1] Devoradores de estrelas (ID: 1001903, 14 dias restantes)

🎯 Auto-selecionado: Devoradores de estrelas

📁 Saída: Devoradores de estrelas - Andy Weir.epub
🚀 Iniciando download...

📖 Devoradores de estrelas - Andy Weir
   Capítulos: 39 | Recursos: 17

Downloading 39 chapters with 8 workers...

✅ EPUB criado: Devoradores de estrelas - Andy Weir.epub (13.5 MB)
   Capítulos: 39/39 | CSS: 2 | Fontes: 8 | Imagens: 6

🎉 Sucesso!
```

## Troubleshooting

| Erro                 | Solução                                           |
| -------------------- | ------------------------------------------------- |
| 401 Unauthorized     | Token expirado. Gere novo no navegador.           |
| Timeout              | Rede lenta. Verifique sua conexão.                |
| SSL Error            | Necessário para ALB interno AWS. Normal.          |
| EPUB vazio           | Verifique se o livro está alugado e ativo.        |
| Token não encontrado | Cole o token quando solicitado ou delete o cache. |

## Atualização

Se você instalou via `go install`:

```bash
go install github.com/publi0/mec-livros-epub/cmd/mectlivros@latest
```

Se você clonou o repositório:

```bash
git pull origin main
go mod tidy
go build -o mectlivros ./cmd/mectlivros
```

Para verificar versão:

```bash
mectlivros --version
```

## Desenvolvimento

Para contribuir ou trabalhar com o código:

```bash
# Clone o repositório
git clone https://github.com/publi0/mec-livros-epub.git
cd mec-livros-epub

# Instale dependências
go mod download

# Execute testes
go test ./...

# Build
go build -o mectlivros ./cmd/mectlivros

# Lint (opcional, instale golangci-lint)
golangci-lint run ./...
```

## Licença

Uso pessoal. Respeite os termos de uso da plataforma MeC Livros.
