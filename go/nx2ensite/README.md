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

## Instalação

- Você pode baixar o binário direto do repositório:
```
cd /tmp
wget https://raw.githubusercontent.com/dotfob/sysadmin-tools/main/go/nx2ensite/bin/nx2ensite-linux-amd64

```
Checagem de integridade
```
wget https://raw.githubusercontent.com/dotfob/sysadmin-tools/main/go/nx2ensite/bin/nx2ensite-linux-amd64.md5
md5sum -c nx2ensite-linux-amd64.md5
```
Colocando o binário no seu devido lugar
```
chmod +x nx2ensite-linux-amd64
sudo mv nx2ensite-linux-amd64 /usr/local/bin/nx2ensite
```
Agora o comando está disponível para você:
```
nx2ensite
```
- ou pode compilar com Go:

Baixe o arquivo nx2ensite.go para um diretório local 
```
cd /tmp
go build -o nx2ensite nx2ensite.go
chmod +x nx2ensite
sudo mv nx2ensite /usr/local/bin/
```




