---
name: "KI-Hebel-Check — RKS"
password: "rks-check"
form_id: "81GYAY"
---

Damit wir unsere Empfehlungen für RKS gezielt vorbereiten können, haben wir abhängig von Ihren Antworten bis zu 31 Fragen zusammengestellt — Dauer: ca. 20 Minuten. Wir freuen uns, wenn mehrere Personen den Fragebogen ausfüllen — jede Einsendung wird separat gespeichert. Bei den offenen Fragen zählt nicht die perfekte Antwort — Ihre erste Eingebung ist oft die wertvollste.

## KI in Ihrem Unternehmen

KI-Standortbestimmung — ca. 2 Minuten.

F1: In welcher Rolle sind Sie im Unternehmen?
> type: single-choice
> required: true
- Geschäftsführung
- Bereichs- oder Abteilungsleitung
- Teamleitung
- Operative(r) Mitarbeiter(in)
- Andere {other}

F2: Welche Rolle spielt KI aktuell in Ihrem Unternehmen?
> type: single-choice
- KI ist strategisch verankert und wird systematisch eingesetzt
- KI ist in einzelnen Prozessen produktiv im Einsatz
- Wir experimentieren gezielt mit KI-Tools für bestimmte Aufgaben
- Einzelne Mitarbeiter nutzen ChatGPT, Copilot o.ä. auf eigene Initiative
- Wir möchten KI einsetzen, wissen aber noch nicht wie
- Keine — KI ist bei uns kein Thema

> show F3, F4 when F2 is_not_any_of "Keine — KI ist bei uns kein Thema", "Wir möchten KI einsetzen, wissen aber noch nicht wie"

F3: Gibt es in Ihrem Unternehmen Regeln oder Absprachen, welche KI-Tools für welche Aufgaben genutzt werden dürfen?
> type: single-choice
> hidden: true
- Ja, wir haben klare Richtlinien, die allen bekannt sind
- Es gibt informelle Absprachen, aber nichts Schriftliches
- Nein, jeder entscheidet selbst
- Das Thema hat sich bei uns noch nicht gestellt

F4: Werden in Ihrem Unternehmen KI-Tools auch mit Firmendaten genutzt — zum Beispiel Kundennamen, Kalkulationen oder interne Dokumente?
> type: single-choice
> hidden: true
- Ja, das kommt regelmäßig vor
- Vermutlich, aber wir haben keinen Überblick
- Nein, das ist bei uns klar geregelt
- Das kann ich nicht einschätzen

---

## Ihr Arbeitsalltag

Nicht Technik, sondern Alltag: Wo binden wiederkehrende Aufgaben wertvolle Kapazität? Ca. 7 Minuten.

F5: Wo im Unternehmen würden Sie sich wünschen, schneller ein klares Bild zu haben?
> type: long-text
> hint: "Zum Beispiel: Wie entwickeln sich die Aufwände bei Pauschal- und Festpreisprojekten? Welche Gutachten sind überfällig? Wie ist die Auslastung der Mitarbeiter über alle Projekte hinweg?"

F6: Was ist die nervigste Routineaufgabe, die Ihre klügsten Köpfe bindet?
> type: long-text

F7: Welche drei dieser Tätigkeiten kosten in Ihrem Unternehmen die meiste Zeit?
> type: multi-choice
> max: 3
- Daten abtippen oder übertragen — *z.B. E-Mails, PDFs oder Formulare ins System einpflegen*
- Informationen zusammentragen — *z.B. Daten aus verschiedenen Systemen zusammensuchen*
- Ähnliche Dokumente neu schreiben — *z.B. Gutachten, Stellungnahmen, Ausschreibungstexte*
- Dokumente oder Vorlagen suchen — *z.B. frühere Gutachten, Referenzprojekte, Vorlagen oder Urteile finden*
- Meetings protokollieren — *z.B. Gesprächsergebnisse nachbereiten und verteilen*
- Daten in mehreren Systemen pflegen — *z.B. Projekt- oder Auftragsdaten synchron halten*
- Anfragen manuell zuordnen — *z.B. eingehende Belege, Aufträge oder Anfragen weiterleiten*
- Freigaben einholen und nachverfolgen — *z.B. Bestellanforderungen, Rechnungsfreigaben*

F8: Stellen Sie sich vor, die zeitaufwändigste dieser Aufgaben würde ab morgen automatisch laufen — was würden Ihre Mitarbeiter stattdessen tun?
> type: long-text

F9: Welche Aufgabe erledigen viele Ihrer Mitarbeiter regelmäßig auf die gleiche Weise — und könnten dabei durch bessere Werkzeuge unterstützt werden?
> type: long-text
> hint: "Zum Beispiel Projektberichte, Kostendokumentation, Nachtragsaufstellungen oder Terminpläne. Uns interessiert nicht, ob die Aufgabe nötig ist — sondern ob sie schneller oder einfacher gehen könnte."

F10: Wie gut funktioniert der Informationsfluss zwischen Ihren Projektteams — zum Beispiel wenn Erfahrungen aus einem Projekt für ein anderes relevant wären?
> type: single-choice
- Sehr gut — Erfahrungen und Informationen fließen schnell zwischen Projekten
- Grundsätzlich gut, aber zwischen einzelnen Projekten gibt es Reibungsverluste
- Schwierig — viel läuft über persönliche Kanäle, Meetings oder Nachfragen
- Es gibt kaum Austausch zwischen Projekten — Informationen bleiben beim jeweiligen Projektteam

F11: Wie werden Ergebnisse aus Baubesprechungen, Ortsterminen oder internen Übergaben bei Ihnen festgehalten?
> type: single-choice
- Digital und zeitnah — Ergebnisse stehen noch am selben Tag bereit
- Manuell mit Verzögerung — Protokolle oder Notizen werden nachträglich erstellt
- Gar nicht — Ergebnisse bleiben im Gedächtnis der Beteiligten
- Unterschiedlich je nach Anlass

F12: Wie aufwändig ist in Ihrem Unternehmen das Erstellen der folgenden Dokumente?
> type: matrix
> columns: Gering, Moderat, Hoch, Nicht relevant
- Regelmäßige Berichte (z.B. Statusberichte, Kostenauswertungen)
- Protokolle (z.B. aus Baubesprechungen, Ortsterminen, Abnahmen)
- Gutachten, Stellungnahmen, Leistungsverzeichnisse

---
> button: "Weiter zu Seite 3 / 5"

## Ihre Systeme

Drei kurze Fragen zu Ihrer Systemlandschaft. *Alle Fragen optional.*

F13: Wie schnell finden Ihre Mitarbeiter die Informationen, die sie für ihre tägliche Arbeit brauchen?
> type: single-choice
> required: false
- Informationen sind schnell und einfach verfügbar
- Meistens ja, aber manche Dinge sind schwer zu finden
- Es dauert oft lange, die richtigen Informationen zusammenzubekommen
- Viele Informationen stecken in den Köpfen einzelner Kollegen

F14: Wo liegen Ihre wichtigsten Geschäftsdaten — zum Beispiel Projekt-, Auftrags- oder Kostendaten?
> type: multi-choice
> required: false
- Eigene Fachanwendungen (z.B. PIS, CoCo)
- Excel-Listen und Tabellen
- E-Mail-Postfächer
- Netzlaufwerke (NAS)
- RKS Cloud (Nextcloud)
- Papierarchive
- Andere {other}

F15: Gibt es ein System oder Werkzeug in Ihrem Unternehmen, das Ihre Mitarbeiter am meisten ausbremst?
> type: long-text
> required: false
> hint: "Zum Beispiel ein umständliches Programm, fehlende Werkzeuge für wiederkehrende Aufgaben, oder der Umweg über Papier und E-Mail, weil ein passendes System fehlt."

---
> button: "Weiter zu Seite 4 / 5"

## Chancen und nächste Schritte

Wo sehen Sie die größten Chancen — und was sind die Rahmenbedingungen? Ca. 8 Minuten.

F16: Wenn Sie eine Sache in Ihrem Unternehmen auf Knopfdruck verbessern könnten — was wäre das?
> type: long-text
> hint: "Beschreiben Sie frei — egal ob groß oder klein, technisch oder organisatorisch."

> show F17 when F16 is_not_empty

F17: Was wäre der nächste Schritt, um diese Verbesserung anzugehen?
> type: single-choice
> hidden: true
- Wir wissen, was zu tun wäre — es fehlt die Kapazität dafür
- Wir bräuchten einen konkreten Plan, wie man das angeht
- Das wurde schon versucht — bisher ohne den gewünschten Erfolg
- Das kann ich nicht beurteilen

F18: Wenn intern alles rund laufen würde — was würde sich für Ihre Auftraggeber spürbar verbessern?
> type: multi-choice
- Schnellere Reaktionszeiten
- Bessere Erreichbarkeit
- Weniger Fehler
- Aktuellere Informationen und Auskünfte
- Persönlichere Betreuung
- Zuverlässigere Terminzusagen
- Andere {other}

F19: Welche strategischen Prioritäten beschäftigen Ihr Unternehmen aktuell?
> type: multi-choice
- Wachstum (neue Standorte, mehr Personal, neue Märkte)
- Fachkräftemangel
- Kostendruck und Effizienz
- KI und Automatisierung
- Digitalisierung bestehender Prozesse
- Neue gesetzliche Anforderungen
- Neue Produkte oder Geschäftsmodelle
- Keines davon

> show F20 when F19 is_not_empty and F19 does_not_contain "Keines davon"

F20: Was steht konkret an?
> type: long-text
> required: false
> hidden: true
> hint: "Beschreiben Sie kurz, was aktuell geplant ist."

F21: Wie würden Sie die Veränderungsbereitschaft in Ihrem Unternehmen beschreiben — wenn es um neue Tools oder veränderte Arbeitsweisen geht?
> type: single-choice
- Hoch — unser Team ist offen für Neues und probiert gerne aus
- Grundsätzlich offen, aber es braucht gute Argumente und Begleitung
- Gemischt — manche ziehen mit, andere bremsen
- Eher zurückhaltend — Veränderungen stoßen auf Widerstand

F22: Was war Ihr letztes Veränderungsprojekt, und wie hat Ihr Team reagiert?
> type: long-text
> required: false
> hint: "Zum Beispiel eine neue Software, ein neuer Prozess oder eine Umstrukturierung."

F23: Wer wäre in Ihrem Unternehmen Ansprechpartner, wenn es um die Umsetzung technischer Verbesserungen geht?
> type: multi-choice
- Externe IT-Dienstleister
- Fachabteilung selbst
- Geschäftsführung direkt
- Unklar

F24: Wie läuft bei Ihnen die Qualitätssicherung von Sachverständigengutachten — von der Rohfassung bis zur Freigabe durch den Sachverständigen?
> type: single-choice
- Systematisch — es gibt eine definierte Prüfkette mit klaren Prüfschritten
- Teilweise standardisiert — einzelne Prüfschritte gibt es, aber vieles hängt von der Sorgfalt des Einzelnen ab
- Überwiegend manuell und personenabhängig — der Sachverständige prüft selbst
- Das kann ich nicht beurteilen

F25: Wie werden Ihre projektbezogenen Dokumentationen — zum Beispiel Nachtragsaufstellungen, Baustellenberichte oder Beweissicherungen — erstellt und weiterverarbeitet?
> type: single-choice
- Digital und strukturiert — Vorlagen und standardisierte Abläufe decken den Großteil ab
- Teilweise standardisiert — Grundstruktur existiert, aber viel individuelle Nacharbeit nötig
- Überwiegend manuell — jedes Dokument wird weitgehend individuell erstellt
- Das kann ich nicht beurteilen

F26: Wie erstellen und pflegen Sie Terminpläne für Ihre Bauvorhaben — und wie flexibel können Sie auf Verzögerungen oder Änderungen reagieren?
> type: single-choice
- Datengestützt mit Planungstool — Abhängigkeiten und kritische Pfade sind jederzeit sichtbar, Änderungen lassen sich schnell durchspielen
- Teilweise toolgestützt — Erstplanung im Tool (z.B. MS Project), aber Anpassungen sind aufwändig und laufen oft manuell
- Überwiegend manuell oder nach Erfahrung — Terminpläne basieren auf Erfahrungswerten, systematische Ablaufsteuerung fehlt
- Das kann ich nicht beurteilen

F27: Wie stellen Sie sicher, dass das Fachwissen Ihrer erfahrenen Sachverständigen und Projektleiter auch für neue Kollegen zugänglich ist — zum Beispiel bei Einarbeitung, Vertretung oder wenn ein Mitarbeiter ausscheidet?
> type: single-choice
- Systematisch — wir haben dokumentierte Vorgehensweisen, Referenzprojekte oder ein internes Wissensarchiv
- Teilweise — einzelne Bereiche sind dokumentiert, aber vieles lebt in den Köpfen der Erfahrenen
- Kaum — Wissen wird mündlich oder über Zusammenarbeit im Projekt weitergegeben
- Das ist bei uns kein großes Thema

F28: Wie aufwändig ist bei Ihnen die Recherche von Urteilen, Normen oder Fachinhalten — zum Beispiel für Gutachten, Stellungnahmen oder Nachtragsmanagement?
> type: single-choice
- Wenig — wir haben gute Zugänge und finden schnell, was wir brauchen
- Moderat — einzelne Recherchen kosten Zeit, sind aber handhabbar
- Hoch — Fachrecherche ist ein wesentlicher Zeitfresser, besonders bei komplexen Sachverhalten
- Das kann ich nicht beurteilen

---
> button: "Absenden"

## Ihre Erwartung an uns

Abschluss — damit wir unsere Empfehlungen auf das ausrichten, was zählt.

F29: Was muss der KI-Hebel-Check liefern, damit er sich gelohnt hat?
> type: long-text

F30: Gibt es sonst noch etwas, das wir wissen sollten?
> type: long-text
> required: false
