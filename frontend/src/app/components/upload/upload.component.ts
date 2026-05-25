import { Component, computed, signal, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { EMPTY, Subject, switchMap, catchError, map, startWith, tap, filter } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSelectModule } from '@angular/material/select';
import { MatDialog } from '@angular/material/dialog';
import { QuizService, Chapter, UploadProgressEvent } from '../../services/quiz.service';
import { ConfirmDialogComponent } from '../confirm-dialog/confirm-dialog.component';

type Provider = 'openai' | 'claude_cli';

@Component({
  selector: 'app-upload',
  standalone: true,
  imports: [
    FormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatListModule,
    MatProgressSpinnerModule,
    MatSelectModule,
  ],
  templateUrl: './upload.component.html',
  styleUrl: './upload.component.scss'
})
export class UploadComponent {

  private quizService = inject(QuizService);
  private router = inject(Router);
  private dialog = inject(MatDialog);

  readonly files = signal<File[]>([]);
  readonly error = signal('');
  readonly isLoading = signal(false);
  readonly progressFile = signal('');
  readonly progressIndex = signal(0);
  readonly progressTotal = signal(0);
  readonly progressStage = signal('');
  readonly progressStartMs = signal(0);
  readonly progressNowMs = signal(0);

  readonly progressElapsedSec = computed(() => {
    const start = this.progressStartMs();
    const now = this.progressNowMs();
    if (!start) return 0;
    return Math.max(0, Math.floor((now - start) / 1000));
  });

  readonly progressStageLabel = computed(() => {
    switch (this.progressStage()) {
      case 'extract':     return 'PDF wird extrahiert';
      case 'chunk':       return 'Text wird zerlegt';
      case 'chunk_done':  return 'Text zerlegt';
      case 'embeddings':  return 'Embeddings werden erzeugt';
      case 'generate':    return 'Fragen werden generiert (kann 30–120s dauern)';
      case 'save':        return 'Fragen werden gespeichert';
      case '':            return '';
      default:            return this.progressStage();
    }
  });

  model = 'gpt-4.1-mini';
  apiKey = '';
  readonly provider = signal<Provider>('openai');

  private readonly refreshTrigger$ = new Subject<void>();
  private readonly apiKeyRefreshTrigger$ = new Subject<void>();
  private readonly providerRefreshTrigger$ = new Subject<void>();

  readonly isOpenAI = computed(() => this.provider() === 'openai');

  readonly hasSavedApiKey = toSignal(
    this.apiKeyRefreshTrigger$.pipe(
      startWith(null),
      switchMap(() =>
        this.quizService.hasSavedApiKey().pipe(
          catchError(() => EMPTY)
        )
      )
    ),
    { initialValue: false }
  );

  private readonly _providerEffect = toSignal(
    this.providerRefreshTrigger$.pipe(
      startWith(null),
      switchMap(() =>
        this.quizService.getProvider().pipe(
          tap(p => {
            const next = (p === 'claude_cli' ? 'claude_cli' : 'openai') as Provider;
            this.provider.set(next);
            if (next === 'claude_cli' && (this.model === '' || this.model.startsWith('gpt-'))) {
              this.model = 'sonnet';
            }
          }),
          catchError(() => EMPTY)
        )
      )
    )
  );

  readonly chapters = toSignal(
    this.refreshTrigger$.pipe(
      startWith(null),
      switchMap(() =>
        this.quizService.getChapters().pipe(
          map((chapters) => Array.isArray(chapters) ? chapters : []),
          catchError(() => EMPTY)
        )
      )
    ),
    { initialValue: [] as Chapter[] }
  );

  readonly chapterList = computed(() => this.chapters() ?? []);

  private elapsedTimer: ReturnType<typeof setInterval> | null = null;

  private startElapsedTimer(): void {
    this.progressStartMs.set(Date.now());
    this.progressNowMs.set(Date.now());
    if (this.elapsedTimer) clearInterval(this.elapsedTimer);
    this.elapsedTimer = setInterval(() => this.progressNowMs.set(Date.now()), 1000);
  }

  private stopElapsedTimer(): void {
    if (this.elapsedTimer) {
      clearInterval(this.elapsedTimer);
      this.elapsedTimer = null;
    }
  }

  private resetProgress(): void {
    this.progressFile.set('');
    this.progressIndex.set(0);
    this.progressTotal.set(0);
    this.progressStage.set('');
    this.progressStartMs.set(0);
    this.progressNowMs.set(0);
    this.stopElapsedTimer();
  }

  private handleProgressEvent(ev: UploadProgressEvent): void {
    switch (ev.event) {
      case 'start':
        this.progressTotal.set(ev.total ?? 0);
        break;
      case 'file_start':
        this.progressFile.set(ev.file ?? '');
        this.progressIndex.set(ev.index ?? 0);
        if (ev.total) this.progressTotal.set(ev.total);
        this.progressStage.set('');
        break;
      case 'stage':
        this.progressStage.set(ev.stage ?? '');
        break;
      case 'file_done':
        this.progressStage.set('');
        break;
      case 'file_error':
        this.error.set(`${ev.file ?? ''}: ${ev.message ?? 'Fehler'}`);
        break;
      case 'error':
        this.error.set(ev.message ?? 'Fehler beim Upload');
        break;
      case 'done':
        // final event; UI is reset by the consumer.
        break;
    }
  }

  private async runStreamingUpload(
    files: File[],
    model: string,
    apiKey: string,
  ): Promise<void> {
    this.isLoading.set(true);
    this.error.set('');
    this.startElapsedTimer();
    try {
      const stream = this.quizService.uploadFilesStreaming(files, model, apiKey);
      for await (const ev of stream) {
        this.handleProgressEvent(ev);
      }
    } catch (err) {
      this.error.set((err as Error).message);
    } finally {
      this.isLoading.set(false);
      this.resetProgress();
      this.files.set([]);
      this.refreshTrigger$.next();
    }
  }

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.files.set(input.files ? Array.from(input.files) : []);
  }

  upload(): void {
    if (this.files().length === 0) {
      this.error.set('Bitte mindestens eine PDF-Datei auswählen.');
      return;
    }
    void this.runStreamingUpload(this.files(), this.model, this.apiKey);
  }

  openQuiz(chapter: Chapter): void {
    this.router.navigate(['/quiz', chapter.id]);
  }

  clearApiKey(): void {
    this.quizService.clearApiKey().pipe(
      catchError((err: Error) => {
        this.error.set(err.message);
        return EMPTY;
      })
    ).subscribe(() => {
      this.apiKeyRefreshTrigger$.next();
    });
  }

  onProviderChange(next: Provider): void {
    this.quizService.setProvider(next).pipe(
      catchError((err: Error) => {
        this.error.set(err.message);
        return EMPTY;
      })
    ).subscribe(() => {
      this.provider.set(next);
      if (next === 'claude_cli' && (this.model === '' || this.model.startsWith('gpt-'))) {
        this.model = 'sonnet';
      }
      if (next === 'openai' && !this.model.startsWith('gpt-')) {
        this.model = 'gpt-4.1-mini';
      }
    });
  }

  deleteChapter(chapter: Chapter): void {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: 'Kapitel löschen',
        message: `Soll das Kapitel "${chapter.title}" wirklich gelöscht werden?`,
      },
    });

    dialogRef.afterClosed().pipe(
      filter((confirmed: boolean) => confirmed),
      switchMap(() =>
        this.quizService.deleteChapter(chapter.id).pipe(
          catchError((err: Error) => {
            this.error.set(err.message);
            return EMPTY;
          })
        )
      )
    ).subscribe(() => {
      this.refreshTrigger$.next();
    });
  }
}
