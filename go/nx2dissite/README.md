# 🔧 nx2dissite - NGINX Disable Site

`nx2dissite` é uma ferramenta escrita em Go que facilita minha vida na habilitação de sites no NGINX, similar ao `a2dissite` do Apache. Funciona de forma bastante simples: remove links simbólicos de arquivos de configuração de sites do diretório `sites-enabled`, testa a configuração do NGINX e recarrega o serviço. Tradução e algumas funções by Grok.

## 📂 Estrutura esperada por padrão

- `/etc/nginx/sites-available/`
- `/etc/nginx/sites-enabled/`

a ferramenta possibilita alterar esse diretório com o parâmetro --config-dir

## 🧪 Exemplo de uso

nx2dissite <site>

 -- avalia se existe o arquivo link simbólico {site}.conf no diretório sites-enabled/
 
 -- remove o link simbólico em sites-enabled/{site}.conf
 
 -- avalia se a configuração está ok, se estiver: faz um reload no nginx

## Instalação

- Você pode baixar o binário direto do repositório:
```
cd /tmp
wget https://raw.githubusercontent.com/dotfob/sysadmin-tools/main/go/nx2dissite/bin/nx2dissite-linux-amd64
```
Checagem de integridade
```
wget https://raw.githubusercontent.com/dotfob/sysadmin-tools/main/go/nx2dissite/bin/nx2dissite-linux-amd64.md5
md5sum -c nx2dissite-linux-amd64.md5
```
Colocando o binário no seu devido lugar
```
chmod +x nx2dissite-linux-amd64
sudo mv nx2dissite /usr/local/bin/nx2dissite
```
Agora o comando está disponível para você:
```
nx2dissite
```
- ou pode compilar com Go:

Baixe o arquivo nx2dissite.go para um diretório local 
```
cd /tmp
go build -o nx2dissite nx2dissite.go
chmod +x nx2dissite
sudo mv nx2dissite /usr/local/bin/
```



