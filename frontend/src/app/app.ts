import { Component, signal, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-root',
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  private http = inject(HttpClient);
  
  protected loading = signal(false);
  protected labUrl = signal<string | null>(null);
  protected errorMsg = signal<string | null>(null);

  startLab() {
    this.loading.set(true);
    this.errorMsg.set(null);
    this.labUrl.set(null);

    const payload = {
      user_id: "f9ad3bb1-f05a-4443-ba73-28a9db44ee60",
      template_id: "1281b9f1-3d25-4d34-b18d-a8c9faab20e9",
      ram_limit_mb: 256,
      user_email: "regulo.rivera@uab.edu.bo"
    };

    const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InJlZ3Vsby5yaXZlcmFAdWFiLmVkdS5ibyIsImV4cCI6MTc4NDc0NTg4Nywicm9sZSI6InN0dWRlbnQiLCJ1c2VyX2lkIjoiZjlhZDNiYjEtZjA1YS00NDQzLWJhNzMtMjhhOWRiNDRlZTYwIn0.vH2ZPeiCJNOhnYUsShdXw7K9E17fw60XX1vx_zNExsg";

    this.http.post<any>('http://localhost:3000/api/v1/labs/start', payload, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }).subscribe({
      next: (res) => {
        const prefix = payload.user_email.split('@')[0];
        this.labUrl.set(`http://${prefix}-lab.solv.uab.edu.bo`);
        this.loading.set(false);
      },
      error: (err) => {
        console.error(err);
        this.errorMsg.set('Fallo de conexión. Verifica que el backend de Go esté corriendo en el puerto 3000 y tenga CORS habilitado.');
        this.loading.set(false);
      }
    });
  }
}
