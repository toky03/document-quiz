import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

export interface Chapter {
  id: number;
  title: string;
  source_name: string;
  question_count: number;
}

export interface QuizQuestion {
  question: string;
  quiz_type: 'single' | 'multiple';
  options: string[];
  correct_options: number[];
  answer: string;
}

export interface QuizResult {
  correct_count: number;
  total_count: number;
  results: Array<{
    index: number;
    question: string;
    user_answer: number[];
    correct_answer: number[];
    is_correct: boolean;
    options: string[];
    quiz_type: string;
  }>;
}

export interface UploadProgressEvent {
  event: 'start' | 'file_start' | 'stage' | 'file_done' | 'file_error' | 'done' | 'error';
  file?: string;
  index?: number;
  total?: number;
  stage?: string;
  message?: string;
  chunk_count?: number;
  generated_pairs?: number;
  result?: unknown;
}

@Injectable({ providedIn: 'root' })
export class QuizService {

  private apiUrl = 'http://localhost:8080/api';

  constructor(private http: HttpClient) {}

  /**
   * Streaming upload: yields one NDJSON event per backend stage.
   * Throws on HTTP/network errors. The stream ends after a 'done' or 'error' event.
   */
  async *uploadFilesStreaming(
    files: File[],
    model: string,
    apiKey: string,
  ): AsyncGenerator<UploadProgressEvent, void, void> {
    const formData = new FormData();
    for (const file of files) {
      formData.append('files', file, file.name);
    }
    formData.append('model', model);
    if (apiKey.trim()) {
      formData.append('api_key', apiKey.trim());
    }

    const res = await fetch(`${this.apiUrl}/upload`, {
      method: 'POST',
      body: formData,
    });

    if (!res.ok || !res.body) {
      throw new Error(`Upload fehlgeschlagen (HTTP ${res.status})`);
    }

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });

      let nl: number;
      while ((nl = buffer.indexOf('\n')) !== -1) {
        const line = buffer.slice(0, nl).trim();
        buffer = buffer.slice(nl + 1);
        if (!line) continue;
        try {
          yield JSON.parse(line) as UploadProgressEvent;
        } catch {
          // Skip malformed lines rather than aborting the whole upload.
        }
      }
    }

    const tail = buffer.trim();
    if (tail) {
      try {
        yield JSON.parse(tail) as UploadProgressEvent;
      } catch {
        // ignore
      }
    }
  }

  getChapters(): Observable<Chapter[]> {
    return this.http.get<Chapter[]>(`${this.apiUrl}/chapters`);
  }

  getQuestions(chapterId: number): Observable<QuizQuestion[]> {
    return this.http.get<QuizQuestion[]>(`${this.apiUrl}/chapters/${chapterId}/questions`);
  }

  submitQuiz(chapterId: number, answers: number[][]): Observable<QuizResult> {
    return this.http.post<QuizResult>(`${this.apiUrl}/quiz/submit`, {
      chapter_id: chapterId,
      answers
    });
  }

  deleteChapter(id: number): Observable<void> {
    return this.http.delete<void>(`${this.apiUrl}/chapters/${id}`);
  }

  hasSavedApiKey(): Observable<boolean> {
    return this.http.get<{ is_saved: boolean }>(`${this.apiUrl}/settings/openai-key-status`).pipe(
      map(res => res.is_saved)
    );
  }

  clearApiKey(): Observable<void> {
    return this.http.delete<void>(`${this.apiUrl}/settings/openai-key`);
  }

  getProvider(): Observable<string> {
    return this.http.get<{ provider: string }>(`${this.apiUrl}/settings/provider`).pipe(
      map(res => res.provider)
    );
  }

  setProvider(provider: string): Observable<unknown> {
    return this.http.post(`${this.apiUrl}/settings/provider`, { provider });
  }
}
