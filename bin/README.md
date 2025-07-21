# 🔧 nx2ensite

`nx2ensite` é uma ferramenta escrita em Go que facilita a habilitação de sites no NGINX, similar ao `a2ensite` do Apache. Ela cria links simbólicos de arquivos de configuração de sites do diretório `sites-available` para `sites-enabled`, testa a configuração do NGINX e recarrega o serviço.

## 📂 Estrutura esperada por padrão

- `/etc/nginx/sites-available/`
- `/etc/nginx/sites-enabled/`

a ferramenta possibilita alterar esse diretório com o parâmetro -f

## 🧪 Exemplo de uso

nx2ensite <site>

 -- avalia se existe o arquivo {site}.conf no diretório sites-available
 -- cria um link simbólico em sites-enabled/{site}.conf apontando para sites-available/{site}.conf
 -- avalia se a configuração está ok, se estiver: faz um reload no nginx


