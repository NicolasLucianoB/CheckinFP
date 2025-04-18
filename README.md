# CheckinFP Backend - API for Volunteer Check-in

**CheckinFP Backend** is a RESTful API built with **Go** using the **Gin Gonic** framework. It is responsible for generating unique QR codes, registering volunteer check-ins, and managing the data related to volunteer attendance.

## 🚀 Features

✅ User authentication with JWT tokens.  
✅ Generate unique QR codes for each volunteer.  
✅ Register check-ins when a volunteer scans their QR code.  
✅ Store data like name and check-in time in the database.  
✅ Easily extensible for future improvements, such as Power BI integration for reports.

## 🛠 Technologies Used

- **Go** (Gin Gonic framework for the API)
- **SQLite** (local database for storing data)
- **JWT** (JSON Web Tokens for secure authentication)
- **Gin Middleware** (for authentication and logging)
- **QR Code Generator** (`github.com/skip2/go-qrcode`)

## 📦 How to Run the Project Locally

Follow these steps to get the backend up and running locally:

### 1. Clone the Repository

```sh
git clone https://github.com/your-username/CheckinFP.git
cd CheckinFP/checkin-backend
```

### 2. Install Dependencies

Make sure you have Go installed and set up:

```sh
go mod tidy
```

### 3. Run the Server

Start the Go server with:

```sh
go run main.go
```

The server will be running at `http://localhost:8080`.

### 4. API Endpoints

- **POST /login**: Logs a user in and returns a JWT token.
- **GET /generate/{volunteer_name}**: Generates a QR code for the specified volunteer.
- **GET /checkin/{volunteer_name}**: Registers the volunteer's check-in when the QR code is scanned.
- **GET /me**: Validates the current user's JWT token and returns user information.

### 5. Testing Locally

1. Generate a QR code for a volunteer:
   ```sh
   http://localhost:8080/generate/VolunteerName
   ```
2. Register a check-in by visiting:
   ```sh
   http://localhost:8080/checkin/VolunteerName
   ```

## 🛠 Planned Improvements

- 📌 Switch from SQLite to MySQL for better scalability.
- 📌 Implement better user input validation.
- 📌 Build a dashboard with data analytics and visualizations (Power BI).
- 📌 Add error handling and more detailed logging.

## 🌐 Deploy

To deploy this backend application, you can use cloud platforms such as **Heroku**, **AWS**, or **DigitalOcean**.

---

📌 **Project Status**: *Under development 🚧*

💡 **Contributions are welcome!**
