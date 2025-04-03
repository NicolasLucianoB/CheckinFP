# 📌 CheckinFP

**CheckinFP** é um sistema de **check-in via QR Code** para monitorar a frequência e horário de chegada dos voluntários da igreja Família Plena.

## 🚀 Funcionalidades

✅ Geração de QR Code único para cada voluntário.  
✅ Registro de check-in ao escanear o QR Code.  
✅ Armazenamento de dados como nome e horário de chegada.  
✅ API em Go para gerenciamento de check-ins.  
✅ Testes locais e integração futura com Power BI para análise de dados.  

## 🛠 Tecnologias Utilizadas

- **Go** (Gin Gonic para a API)
- **SQLite** (Banco de dados local)
- **QR Code Generator** (`github.com/skip2/go-qrcode`)
- **Power BI** (para relatórios e dashboards futuros)

## 📦 Como Rodar o Projeto Localmente

1. **Clone o repositório**  
   ```sh
   git clone https://github.com/NicolasLucianoB/CheckinFP.git
   cd CheckinFP
   ```

2. **Instale as dependências**  
   ```sh
   go mod tidy
   ```

3. **Execute o servidor**  
   ```sh
   go run main.go
   ```

4. **Gerar um QR Code para um voluntário**  
   Acesse no navegador:  
   ```
   http://SEU-IP-LOCAL:8080/generate/NOME_DO_VOLUNTARIO
   ```
   _(Substitua `SEU-IP-LOCAL` pelo IP da sua máquina e `NOME_DO_VOLUNTARIO` pelo nome do voluntário.)_

5. **Fazer check-in escaneando o QR Code**  
   O QR Code gerado conterá um link no formato:  
   ```
   http://SEU-IP-LOCAL:8080/checkin/NOME_DO_VOLUNTARIO
   ```
   Ao acessar esse link via celular, o check-in será registrado.

## 🛠 Melhorias Planejadas

- 📌 Criar interface para cadastro de voluntários.  
- 📌 substituir o banco de dados utilizado para MySQL.  
- 📌 Exibir mensagem de sucesso ao invés de JSON no check-in.  
- 📌 Criar um dashboard no Power BI com dados de assiduidade.  
- 📌 Implementar ranking de desempenho dos voluntários.  
- 📌 Melhorar a experiência visual do usuário.  

---

📌 **Status do projeto:** *Em desenvolvimento 🚧*  

💡 **Contribuições e sugestões são bem-vindas!**  
