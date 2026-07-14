import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { AuctionService } from '../auction.service';
import { Auction } from '../auction.model';

// Fake bidder for the bidding slice: identity/auth is a separate slice. Distinct
// from the fake seller so the future shill-bidding rule (bidder != seller) holds.
const FAKE_BIDDER_ID = '22222222-2222-2222-2222-222222222222';

@Component({
  selector: 'app-auction-detail',
  imports: [RouterLink, DatePipe, ReactiveFormsModule],
  templateUrl: './auction-detail.html',
})
export class AuctionDetail implements OnInit {
  private svc = inject(AuctionService);
  private route = inject(ActivatedRoute);
  private fb = inject(FormBuilder);

  auction = signal<Auction | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);

  bidding = signal(false);
  bidError = signal<string | null>(null);

  // Current price: the highest bid so far, or the start price if there are none.
  currentPrice = computed(() => {
    const a = this.auction();
    if (!a) return 0;
    return a.highestBidAmount ?? a.startPrice;
  });

  // Smallest amount the next bid may have.
  minNextBid = computed(() => {
    const a = this.auction();
    if (!a) return 0;
    return a.highestBidAmount === null ? a.startPrice : a.highestBidAmount + a.minIncrement;
  });

  form = this.fb.nonNullable.group({
    amount: [0, [Validators.required, Validators.min(1)]],
  });

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      this.error.set('Subasta inválida.');
      this.loading.set(false);
      return;
    }
    this.svc.get(id).subscribe({
      next: (a) => {
        this.auction.set(a);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err?.status === 404 ? 'Subasta no encontrada.' : 'No se pudo cargar la subasta.');
        this.loading.set(false);
      },
    });
  }

  placeBid(): void {
    const a = this.auction();
    if (!a || this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }
    this.bidding.set(true);
    this.bidError.set(null);

    this.svc.placeBid(a.id, { bidderId: FAKE_BIDDER_ID, amount: this.form.getRawValue().amount }).subscribe({
      next: (res) => {
        this.auction.set(res.auction); // refresh current price + highest bidder
        this.form.reset({ amount: 0 });
        this.bidding.set(false);
      },
      error: (err) => {
        this.bidError.set(this.bidErrorMessage(err));
        this.bidding.set(false);
      },
    });
  }

  // Maps the server's response to a message. 409 = state conflict (too low /
  // closed), 503 = the row was busy and the client should retry.
  private bidErrorMessage(err: { status?: number; error?: { error?: string } }): string {
    switch (err?.status) {
      case 409:
        return err.error?.error ?? 'La puja no supera el mínimo o la subasta cerró.';
      case 503:
        return 'La subasta está recibiendo muchas pujas. Intentá de nuevo.';
      case 404:
        return 'La subasta ya no existe.';
      default:
        return 'No se pudo registrar la puja.';
    }
  }

  formatCLP(amount: number): string {
    return new Intl.NumberFormat('es-CL', { style: 'currency', currency: 'CLP' }).format(amount);
  }
}
