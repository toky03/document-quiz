import { Component, signal, inject, computed, effect } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { forkJoin, of, catchError } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatRadioModule } from '@angular/material/radio';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { QuizService, QuizQuestion } from '../../services/quiz.service';

interface DisasterEntry {
  key: string;
  chapterId: number;
  question: QuizQuestion;
}

const TARGET_STREAK = 2;

@Component({
  selector: 'app-disaster',
  standalone: true,
  imports: [
    FormsModule,
    RouterLink,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatRadioModule,
    MatCheckboxModule,
    MatChipsModule,
    MatProgressBarModule,
  ],
  templateUrl: './disaster.component.html',
  styleUrl: './disaster.component.scss',
})
export class DisasterComponent {
  private route = inject(ActivatedRoute);
  private quizService = inject(QuizService);

  readonly chapterIds: number[] = this.parseChapters();
  private readonly storageKey = `disaster:${[...this.chapterIds].sort((a, b) => a - b).join(',')}`;

  readonly error = signal(this.chapterIds.length === 0 ? 'Keine Kapitel ausgewählt.' : '');
  readonly isLoading = signal(this.chapterIds.length > 0);
  readonly entries = signal<DisasterEntry[]>([]);
  readonly streaks = signal<Record<string, number>>({});
  readonly currentKey = signal<string | null>(null);
  readonly selectedOptions = signal<number[]>([]);
  readonly resultRevealed = signal(false);
  readonly gameDone = signal(false);

  private readonly entryMap = computed(() => {
    const m = new Map<string, DisasterEntry>();
    for (const e of this.entries()) m.set(e.key, e);
    return m;
  });
  readonly currentEntry = computed(() => {
    const k = this.currentKey();
    return k ? this.entryMap().get(k) ?? null : null;
  });
  readonly totalQuestions = computed(() => this.entries().length);
  readonly completedCount = computed(() => {
    const s = this.streaks();
    return this.entries().filter(e => (s[e.key] ?? 0) >= TARGET_STREAK).length;
  });
  readonly remainingCount = computed(() => this.totalQuestions() - this.completedCount());
  readonly progressPercent = computed(() => {
    const t = this.totalQuestions();
    return t === 0 ? 0 : (this.completedCount() / t) * 100;
  });
  readonly currentStreak = computed(() => {
    const k = this.currentKey();
    if (!k) return 0;
    return this.streaks()[k] ?? 0;
  });
  readonly lastWasCorrect = computed(() => {
    const e = this.currentEntry();
    if (!e || !this.resultRevealed()) return false;
    return this.isAnswerCorrect(this.selectedOptions(), e.question.correct_options);
  });

  constructor() {
    if (this.chapterIds.length === 0) {
      this.isLoading.set(false);
    } else {
      forkJoin(
        this.chapterIds.map(id =>
          this.quizService.getQuestions(id).pipe(
            catchError(() => of([] as QuizQuestion[])),
          ),
        ),
      ).subscribe({
        next: (results) => {
          const flat: DisasterEntry[] = [];
          results.forEach((qs, i) => {
            const chapterId = this.chapterIds[i];
            qs.forEach((question, idx) => {
              flat.push({ key: `${chapterId}:${idx}`, chapterId, question });
            });
          });
          if (flat.length === 0) {
            this.error.set('Die ausgewählten Kapitel enthalten keine Fragen.');
            this.isLoading.set(false);
            return;
          }
          this.entries.set(flat);
          this.loadProgress(flat);
          this.isLoading.set(false);
          this.pickNext();
        },
        error: (err: Error) => {
          this.error.set(err.message);
          this.isLoading.set(false);
        },
      });
    }

    effect(() => {
      const s = this.streaks();
      if (this.entries().length === 0) return;
      try {
        localStorage.setItem(this.storageKey, JSON.stringify(s));
      } catch { /* ignore quota / privacy mode */ }
    });
  }

  private parseChapters(): number[] {
    const raw = this.route.snapshot.queryParamMap.get('chapters') ?? '';
    return raw.split(',')
      .map(x => Number(x))
      .filter(x => Number.isFinite(x) && x > 0);
  }

  private loadProgress(entries: DisasterEntry[]): void {
    try {
      const raw = localStorage.getItem(this.storageKey);
      if (!raw) return;
      const stored = JSON.parse(raw) as Record<string, number>;
      const valid: Record<string, number> = {};
      const validKeys = new Set(entries.map(e => e.key));
      for (const [k, v] of Object.entries(stored)) {
        if (validKeys.has(k) && typeof v === 'number' && v >= 0 && v <= TARGET_STREAK) {
          valid[k] = v;
        }
      }
      this.streaks.set(valid);
    } catch { /* ignore */ }
  }

  private pickNext(): void {
    const s = this.streaks();
    const pool = this.entries().filter(e => (s[e.key] ?? 0) < TARGET_STREAK);
    if (pool.length === 0) {
      this.gameDone.set(true);
      this.currentKey.set(null);
      return;
    }
    const picked = pool[Math.floor(Math.random() * pool.length)];
    this.currentKey.set(picked.key);
    this.selectedOptions.set([]);
    this.resultRevealed.set(false);
  }

  toggleOption(optionIndex: number, quizType: 'single' | 'multiple'): void {
    if (this.resultRevealed()) return;
    this.selectedOptions.update(current => {
      if (quizType === 'single') return [optionIndex];
      const idx = current.indexOf(optionIndex);
      return idx >= 0 ? current.filter((_, i) => i !== idx) : [...current, optionIndex];
    });
  }

  isChecked(optionIndex: number): boolean {
    return this.selectedOptions().includes(optionIndex);
  }

  hasSelection(): boolean {
    return this.selectedOptions().length > 0;
  }

  isAnswerCorrect(user: number[], correct: number[]): boolean {
    if (user.length !== correct.length) return false;
    const a = [...user].sort((x, y) => x - y);
    const b = [...correct].sort((x, y) => x - y);
    return a.every((v, i) => v === b[i]);
  }

  submit(): void {
    const e = this.currentEntry();
    if (!e || !this.hasSelection() || this.resultRevealed()) return;
    const correct = this.isAnswerCorrect(this.selectedOptions(), e.question.correct_options);
    this.streaks.update(prev => ({
      ...prev,
      [e.key]: correct ? (prev[e.key] ?? 0) + 1 : 0,
    }));
    this.resultRevealed.set(true);
  }

  continueNext(): void {
    this.pickNext();
  }

  restart(): void {
    this.streaks.set({});
    try { localStorage.removeItem(this.storageKey); } catch { /* ignore */ }
    this.gameDone.set(false);
    this.pickNext();
  }

  isCorrectlySelectedOption(user: number[], correct: number[], idx: number): boolean {
    return user.includes(idx) && correct.includes(idx);
  }
  isWronglySelectedOption(user: number[], correct: number[], idx: number): boolean {
    return user.includes(idx) && !correct.includes(idx);
  }
  isMissedCorrectOption(user: number[], correct: number[], idx: number): boolean {
    return !user.includes(idx) && correct.includes(idx);
  }
}
