import { Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuctionService } from '../auction.service';
import { CreateAuctionRequest } from '../auction.model';

// Fake seller for slice 1: identity/auth is a separate slice. seller_id is a
// well-typed UUID with no FK yet, so wiring real users later is cheap.
const FAKE_SELLER_ID = '11111111-1111-1111-1111-111111111111';

@Component({
  selector: 'app-auction-create',
  imports: [ReactiveFormsModule, RouterLink],
  templateUrl: './auction-create.html',
})
export class AuctionCreate {
  private fb = inject(FormBuilder);
  private svc = inject(AuctionService);
  private router = inject(Router);

  submitting = signal(false);
  serverError = signal<string | null>(null);

  form = this.fb.nonNullable.group({
    title: ['', Validators.required],
    description: [''],
    category: [''],
    startPrice: [0, [Validators.required, Validators.min(0)]],
    minIncrement: [1000, [Validators.required, Validators.min(1)]],
    cap: [null as number | null],
    endsAt: ['', Validators.required],
  });

  submit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }
    this.submitting.set(true);
    this.serverError.set(null);

    const v = this.form.getRawValue();
    const req: CreateAuctionRequest = {
      sellerId: FAKE_SELLER_ID,
      title: v.title,
      description: v.description,
      category: v.category,
      startPrice: v.startPrice,
      minIncrement: v.minIncrement,
      cap: v.cap ? Number(v.cap) : null,
      endsAt: new Date(v.endsAt).toISOString(),
    };

    this.svc.create(req).subscribe({
      next: (a) => this.router.navigate(['/auctions', a.id]),
      error: (err) => {
        this.serverError.set(err?.error?.error ?? 'No se pudo crear la subasta.');
        this.submitting.set(false);
      },
    });
  }
}
