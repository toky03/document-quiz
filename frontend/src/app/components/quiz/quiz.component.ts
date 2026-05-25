import { Component, signal, inject, effect, computed } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { EMPTY, Subject, switchMap, catchError } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatRadioModule } from '@angular/material/radio';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { QuizService, QuizQuestion, QuizResult } from '../../services/quiz.service';

@Component({
  selector: 'app-quiz',
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
  templateUrl: './quiz.component.html',
  styleUrl: './quiz.component.scss'
})
export class QuizComponent {
  private route = inject(ActivatedRoute);
  private quizService = inject(QuizService);

  readonly chapterId = Number(this.route.snapshot.paramMap.get('chapterId'));
  readonly error = signal(this.chapterId > 0 ? '' : 'Ungültige Kapitel-ID.');

  readonly questions = toSignal(
    this.quizService.getQuestions(this.chapterId).pipe(
      catchError((err: Error) => { this.error.set(err.message); return EMPTY; })
    ),
    { initialValue: [] as QuizQuestion[] }
  );

  readonly answers = signal<number[][]>([]);
  readonly currentQuestionIndex = signal(0);
  // displayOrder[i] is the canonical index of the question shown at
  // position i. Lets us shuffle the play order without renaming
  // anything else: `answers`, `revealedIndices`, etc. all remain keyed
  // by display position; we only translate to canonical on submit.
  readonly displayOrder = signal<number[]>([]);
  // optionDisplayOrder[displayQIdx] is the permutation of canonical
  // option indices for the question shown at displayQIdx. answers[] /
  // correct_options[] / explanations[] are stored in canonical option
  // order; the template iterates display order but always references
  // the original canonical index, so the rest of the component stays
  // unchanged.
  readonly optionDisplayOrder = signal<number[][]>([]);
  readonly totalQuestions = computed(() => this.questions().length);

  readonly displayedQuestions = computed(() => {
    const qs = this.questions();
    const order = this.displayOrder();
    if (order.length !== qs.length) return qs;
    return order.map(i => qs[i]);
  });

  readonly currentQuestionDisplayedOptions = computed(() => {
    const q = this.currentQuestion();
    if (!q) return [] as Array<{ canonicalIdx: number; text: string; explanation: string }>;
    const order = this.optionDisplayOrder()[this.currentQuestionIndex()];
    const useOrder = order && order.length === q.options.length
      ? order
      : q.options.map((_, i) => i);
    return useOrder.map(canonicalIdx => ({
      canonicalIdx,
      text: q.options[canonicalIdx],
      explanation: q.explanations?.[canonicalIdx] ?? '',
    }));
  });
  readonly allAnswered = computed(() => {
    const questions = this.questions();
    const givenAnswers = this.answers();
    if (questions.length === 0) {
      return false;
    }
    return questions.every((_, index) => (givenAnswers[index] ?? []).length > 0);
  });
  readonly answeredCount = computed(() => this.answers().filter(answer => answer.length > 0).length);

  readonly practiceMode = signal(false);
  readonly revealedIndices = signal<ReadonlySet<number>>(new Set<number>());

  // Once any question has been answered, the mode is fixed for the rest
  // of the session — user picks practice or not before starting.
  readonly practiceLocked = computed(() => this.answeredCount() > 0);

  readonly isCurrentRevealed = computed(() =>
    this.practiceMode() && this.revealedIndices().has(this.currentQuestionIndex())
  );

  readonly practiceScore = computed(() => {
    const displayed = this.displayedQuestions();
    const answers = this.answers();
    const revealed = this.revealedIndices();
    let correct = 0;
    let wrong = 0;
    for (const idx of revealed) {
      const q = displayed[idx];
      if (!q) continue;
      if (this.isAnswerCorrect(answers[idx] ?? [], q.correct_options)) {
        correct++;
      } else {
        wrong++;
      }
    }
    return { correct, wrong, open: displayed.length - correct - wrong };
  });
  readonly progressPercent = computed(() => {
    const total = this.totalQuestions();
    if (total === 0) {
      return 0;
    }
    return (this.answeredCount() / total) * 100;
  });
  readonly currentQuestion = computed(() => {
    const questions = this.displayedQuestions();
    const index = this.currentQuestionIndex();
    return questions[index];
  });

  private readonly submitTrigger$ = new Subject<number[][]>();

  readonly result = toSignal<QuizResult>(
    this.submitTrigger$.pipe(
      switchMap(answers =>
        this.quizService.submitQuiz(this.chapterId, answers).pipe(
          catchError((err: Error) => { this.error.set(err.message); return EMPTY; })
        )
      )
    )
  );

  constructor() {
    effect(() => {
      const questions = this.questions();
      this.answers.set(new Array(questions.length).fill(null).map(() => []));
      this.currentQuestionIndex.set(0);
      this.revealedIndices.set(new Set<number>());
      const order = this.shuffledIndices(questions.length);
      this.displayOrder.set(order);
      // Per-question option permutation, keyed by display question index
      // so it lines up with displayedQuestions / answers / reveals.
      this.optionDisplayOrder.set(
        order.map(canonicalQIdx => this.shuffledIndices(questions[canonicalQIdx]?.options.length ?? 0)),
      );
    });
  }

  private shuffledIndices(n: number): number[] {
    const arr = Array.from({ length: n }, (_, i) => i);
    for (let i = arr.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [arr[i], arr[j]] = [arr[j], arr[i]];
    }
    return arr;
  }

  readonly displayedResults = computed(() => {
    const r = this.result();
    if (!r) return null;
    const order = this.displayOrder();
    if (order.length !== r.results.length) return r;
    return {
      ...r,
      results: order.map(i => r.results[i]),
    };
  });

  readonly displayedResultItems = computed(() => {
    const r = this.displayedResults();
    if (!r) return [];
    const optionOrders = this.optionDisplayOrder();
    return r.results.map((item, displayQIdx) => {
      const optOrder = optionOrders[displayQIdx];
      const useOrder = optOrder && optOrder.length === item.options.length
        ? optOrder
        : item.options.map((_, i) => i);
      const displayOptions = useOrder.map(canonicalIdx => ({
        canonicalIdx,
        text: item.options[canonicalIdx],
        explanation: item.explanations?.[canonicalIdx] ?? '',
      }));
      return { ...item, displayOptions };
    });
  });

  previousQuestion(): void {
    this.currentQuestionIndex.update(index => Math.max(0, index - 1));
  }

  nextQuestion(): void {
    this.currentQuestionIndex.update(index => Math.min(this.totalQuestions() - 1, index + 1));
  }

  isCurrentQuestionAnswered(): boolean {
    return this.isQuestionAnswered(this.currentQuestionIndex());
  }

  isQuestionAnswered(questionIndex: number): boolean {
    return (this.answers()[questionIndex] ?? []).length > 0;
  }

  getQuestionText(question: QuizQuestion): string {
    const candidate = question as QuizQuestion & {
      question_text?: string;
      text?: string;
    };
    const text = candidate.question ?? candidate.question_text ?? candidate.text ?? '';
    const normalized = String(text).trim();
    return normalized.length > 0 ? normalized : 'Fragetext nicht verfugbar';
  }

  toggleOption(questionIndex: number, optionIndex: number, quizType: 'single' | 'multiple'): void {
    this.answers.update(all => {
      const updated = all.map(a => [...a]);
      const current = updated[questionIndex] ?? [];
      if (quizType === 'single') {
        updated[questionIndex] = [optionIndex];
      } else {
        const idx = current.indexOf(optionIndex);
        updated[questionIndex] = idx >= 0
          ? current.filter((_, i) => i !== idx)
          : [...current, optionIndex];
      }
      return updated;
    });
  }

  isChecked(questionIndex: number, optionIndex: number): boolean {
    return (this.answers()[questionIndex] ?? []).includes(optionIndex);
  }

  isAnswerCorrectOption(correctAnswer: number[], optionIndex: number): boolean {
    return correctAnswer.includes(optionIndex);
  }

  isCorrectlySelectedOption(userAnswer: number[], correctAnswer: number[], optionIndex: number): boolean {
    return userAnswer.includes(optionIndex) && correctAnswer.includes(optionIndex);
  }

  isWronglySelectedOption(userAnswer: number[], correctAnswer: number[], optionIndex: number): boolean {
    return userAnswer.includes(optionIndex) && !correctAnswer.includes(optionIndex);
  }

  isMissedCorrectOption(userAnswer: number[], correctAnswer: number[], optionIndex: number): boolean {
    return !userAnswer.includes(optionIndex) && correctAnswer.includes(optionIndex);
  }

  submit(): void {
    if (!this.allAnswered()) {
      this.error.set('Bitte alle Fragen beantworten, bevor ausgewertet wird.');
      return;
    }
    this.error.set('');

    // The backend zips the answers array against questions in canonical
    // (id ASC) order, so translate from display order before sending.
    const order = this.displayOrder();
    const displayed = this.answers();
    const canonical: number[][] = new Array(displayed.length).fill(null).map(() => []);
    for (let i = 0; i < displayed.length; i++) {
      const canonicalIdx = order[i] ?? i;
      canonical[canonicalIdx] = displayed[i];
    }
    this.submitTrigger$.next(canonical);
  }

  reveal(): void {
    if (!this.practiceMode()) return;
    if (!this.isCurrentQuestionAnswered()) return;
    const index = this.currentQuestionIndex();
    this.revealedIndices.update(prev => {
      if (prev.has(index)) return prev;
      const next = new Set(prev);
      next.add(index);
      return next;
    });
  }

  isQuestionRevealed(index: number): boolean {
    return this.practiceMode() && this.revealedIndices().has(index);
  }

  isAnswerCorrect(user: number[], correct: number[]): boolean {
    if (user.length !== correct.length) return false;
    const a = [...user].sort((x, y) => x - y);
    const b = [...correct].sort((x, y) => x - y);
    return a.every((v, i) => v === b[i]);
  }
}
 