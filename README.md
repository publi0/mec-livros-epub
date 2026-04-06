# MeC Livros Ebook Downloader

Aplicativo em Go 1.26 para download de ebooks da plataforma [MeC Livros](https://meclivros.mec.gov.br).

---

## ⚠️ Avisos Importantes

**Segurança do Token JWT**
O token exigido por esta aplicação é uma credencial sensível que concede acesso à sua conta no MeC Livros.

- Nunca compartilhe seu token com terceiros.
- Não publique ou versione o arquivo de cache (`~/.mec_livros_token`) em repositórios (o `.gitignore` já está configurado para preveni-lo).
- Em caso de suspeita de vazamento, encerre suas sessões ativas na plataforma e exclua o cache local.

**Uso e Direitos Autorais**
Esta é uma ferramenta não-oficial desenvolvida para facilitar a leitura offline. A plataforma MeC Livros opera em um modelo de **empréstimo temporário**. Ao utilizar este software, você é inteiramente responsável por cumprir os Termos de Serviço da plataforma, o que inclui:

- Utilizar os arquivos baixados estritamente para leitura pessoal.
- **Excluir os arquivos EPUB** após o término do período de aluguel estipulado pela plataforma.
- Não distribuir, compartilhar, alterar ou comercializar os ebooks, respeitando a Lei de Direitos Autorais (Lei nº 9.610/1998).

**Isenção de Responsabilidade**
Este software é fornecido "como está" (as-is), sem garantias de qualquer tipo. O desenvolvedor não possui nenhum vínculo com o Ministério da Educação (MEC), UFMG ou mantenedores da plataforma MeC Livros, e não se responsabiliza pelo uso indevido da ferramenta ou por infrações aos termos de serviço.

---

## Estrutura

```text
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

1. Informe ou use o token salvo em cache (`~/.mec_livros_token`).
2. O sistema listará seus ebooks alugados ativamente.
3. Selecione qual livro deseja baixar.
4. O download será feito de forma paralela (8 workers).
5. O arquivo EPUB será gerado e salvo para leitura.

Os arquivos EPUB são salvos na pasta `epubs/` (criada automaticamente se não existir).

## Estrutura de Arquivos

```text
mec-livros-epub/
├── cmd/                     # Aplicação CLI
├── internal/                # Pacotes internos
├── pkg/                     # Pacotes públicos
├── epubs/                   # 📁 Arquivos EPUB salvos aqui (criado automaticamente)
├── .mec_livros_token        # Cache JWT (gitignored)
├── README.md
└── go.mod
```

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

- **Workers:** 8
- **Timeout:** 30s por requisição, 10min no total da operação
- **Cache:** `~/.mec_livros_token` (permissão 600 - apenas leitura/escrita para o proprietário)
- **Output:** Diretório `epubs/`

## Recursos Modernos (Go 1.26)

- **slog**: Structured logging nativo
- **context**: Cancellation e timeouts
- **cmp.Or**: Generic comparisons
- **atomic**: Thread-safe counters
- **goroutines**: Download paralelo

## Obter Token JWT

1. Acesse [meclivros.mec.gov.br](https://meclivros.mec.gov.br) e faça login com sua conta gov.br.
2. Abra o DevTools do navegador (F12) e navegue até a aba **Application** > **Local Storage**.
3. Copie o valor do campo `token` ou `jwt`.
4. Cole o token no aplicativo quando solicitado.

## Cache

O token é salvo automaticamente em `~/.mec_livros_token` para reutilização futura. O arquivo é criado com permissões restritas (600) por segurança.

Para limpar o cache manualmente:

```bash
rm ~/.mec_livros_token
```

Ou diretamente pelo prompt do aplicativo:

```text
Usar token do cache? [Y/n/limpar]: limpar
```

## Saída Exemplo

```text
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

✅ EPUB criado: epubs/Devoradores de estrelas - Andy Weir.epub (13.5 MB)
   Capítulos: 39/39 | CSS: 2 | Fontes: 8 | Imagens: 6

🎉 Sucesso!
```

## Troubleshooting

| Erro                 | Solução                                                  |
| -------------------- | -------------------------------------------------------- |
| 401 Unauthorized     | Token expirado. Gere novo no navegador.                  |
| Timeout              | Rede lenta. Verifique sua conexão.                       |
| SSL Error            | Necessário para ALB interno AWS. Comportamento normal.   |
| EPUB vazio           | Verifique se o livro está alugado e ativo na plataforma. |
| Token não encontrado | Cole o token quando solicitado ou delete o cache.        |

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

Uso pessoal. O código fonte deste repositório é livre, mas isso **não confere nenhum direito** sobre os ebooks baixados através da ferramenta. O conteúdo digital pertence aos respectivos autores e editoras.
