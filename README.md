# MeC Livros Ebook Downloader

Aplicativo em Go 1.26 para download de ebooks da plataforma [MeC Livros](https://meclivros.mec.gov.br) e geração de arquivos EPUB.

---

## ⚠️ Avisos Importantes

- **Segurança do Token:** O token JWT exigido por esta aplicação é uma credencial sensível. Nunca o compartilhe, publique ou versione em repositórios. O arquivo de cache (`~/.mec_livros_token`) é criado automaticamente com permissões restritas (600) para sua proteção.
- **Uso e Direitos Autorais:** Esta é uma ferramenta não-oficial para facilitar a leitura offline de **empréstimos temporários**. Você é inteiramente responsável por:
  - Utilizar os arquivos baixados estritamente para leitura pessoal.
  - **Excluir os arquivos EPUB** da pasta `epubs/` após o término do período de aluguel estipulado pela plataforma.
  - Não distribuir, compartilhar ou comercializar os ebooks, respeitando a Lei de Direitos Autorais (Lei nº 9.610/1998).
- **Isenção de Responsabilidade:** Este software é fornecido "como está" (as-is). O desenvolvedor não possui vínculo com o Ministério da Educação (MEC) ou UFMG e não se responsabiliza pelo uso indevido da ferramenta ou infrações aos termos de serviço.

---

## 🚀 Instalação e Atualização

Requer **Go 1.26+**.

**Via `go install` (Recomendado):**

```bash
go install github.com/publi0/mec-livros-epub/cmd/mectlivros@latest
```

_(O binário será instalado em `$GOPATH/bin/` e pode ser atualizado rodando o mesmo comando)._

**Via clone local:**

```bash
git clone https://github.com/publi0/mec-livros-epub.git
cd mec-livros-epub
go build -o mectlivros ./cmd/mectlivros
```

## 📖 Como Usar

### 1. Obtenha seu Token JWT

1. Acesse [meclivros.mec.gov.br](https://meclivros.mec.gov.br) e faça login com sua conta gov.br.
2. Abra o DevTools do navegador (F12) > **Application** > **Local Storage**.
3. Copie o valor do campo `token` ou `jwt`.

### 2. Execute o Downloader

Execute o comando no terminal:

```bash
mectlivros
```

- Cole o token quando solicitado (ele será salvo em cache para os próximos usos).
- O sistema listará seus livros alugados ativamente. Selecione o que deseja baixar.
- O download será feito de forma paralela (8 workers).
- O arquivo EPUB final será salvo na pasta `epubs/` (criada automaticamente no diretório atual).

_Dica: Para limpar o token salvo ou trocar de conta, basta excluir o cache: `rm ~/.mec_livros_token`_

## 🔍 Exemplo de Saída

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

## 🛠️ Solução de Problemas

| Erro             | Solução                                                     |
| ---------------- | ----------------------------------------------------------- |
| 401 Unauthorized | Token expirado. Gere um novo no navegador e limpe o cache.  |
| Timeout          | Rede lenta (timeout padrão de 10min). Verifique a conexão.  |
| SSL Error        | Necessário para ALB interno AWS. Comportamento normal.      |
| EPUB vazio       | Verifique se o livro está ativamente alugado na plataforma. |

## ⚙️ Arquitetura e Recursos Modernos

- **Estrutura:** `cmd/` (CLI principal), `internal/` (Downloader, Cache, EPUB Builder), `pkg/models/` (Data structs).
- **Go 1.26:** Utiliza bibliotecas e padrões modernos do Go, como `slog` para structured logging, `context` para timeout de operações, concorrência nativa (`goroutines` e `atomic` counters) e generics (`cmp.Or`).
- **Dependências Mínimas:** Foco em utilizar a Standard Library o máximo possível.

## 📄 Licença

Uso pessoal. O código fonte deste repositório é livre, mas isso **não confere nenhum direito** sobre os ebooks baixados através da ferramenta. O conteúdo digital pertence aos respectivos autores e editoras.
