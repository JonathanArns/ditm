FROM scratch

COPY fuzznet .

EXPOSE 80

CMD ["./fuzznet"]