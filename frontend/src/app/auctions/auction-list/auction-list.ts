import { Component, OnInit, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { RouterLink } from '@angular/router';
import { AuctionService } from '../auction.service';
import { Auction } from '../auction.model';

@Component({
  selector: 'app-auction-list',
  imports: [RouterLink, DatePipe],
  templateUrl: './auction-list.html',
})
export class AuctionList implements OnInit {
  private svc = inject(AuctionService);

  auctions = signal<Auction[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);

  ngOnInit(): void {
    this.svc.list().subscribe({
      next: (a) => {
        this.auctions.set(a);
        this.loading.set(false);
      },
      error: () => {
        this.error.set('No se pudo cargar el catálogo.');
        this.loading.set(false);
      },
    });
  }

  formatCLP(amount: number): string {
    return new Intl.NumberFormat('es-CL', { style: 'currency', currency: 'CLP' }).format(amount);
  }
}
